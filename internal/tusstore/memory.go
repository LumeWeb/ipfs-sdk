package tusstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tus/tusd/v2/pkg/handler"
)

// MemoryFile represents an in-memory upload
type MemoryFile struct {
	mu         sync.RWMutex
	info       handler.FileInfo
	binPath    string
	store      *MemoryStore
	isComplete bool
}

func (f *MemoryFile) GetInfo(ctx context.Context) (handler.FileInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	return f.info, nil
}

func (f *MemoryFile) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store.mu.Lock()
	defer f.store.mu.Unlock()

	if f.isComplete {
		return 0, os.ErrClosed
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}

	existing, exists := f.store.data[f.info.ID]
	if !exists {
		return 0, os.ErrNotExist
	}

	// Append to the existing data - like filestore's os.O_APPEND behavior
	extended := append(existing, data...)
	f.store.data[f.info.ID] = extended
	f.info.Offset += int64(len(data))

	return int64(len(data)), nil
}

func (f *MemoryFile) Terminate(ctx context.Context) error {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()

	delete(f.store.data, f.info.ID)
	delete(f.store.metadata, f.info.ID)
	return nil
}

func (f *MemoryFile) DeleteInfo(ctx context.Context) error {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()

	delete(f.store.metadata, f.info.ID)
	return nil
}

func (f *MemoryFile) FinishUpload(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.isComplete = true
	f.info.Offset = f.info.Size
	
	return nil
}

func (f *MemoryFile) DeclareLength(ctx context.Context, length int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.info.Size = length
	f.info.SizeIsDeferred = false
	return f.writeInfo()
}

func (f *MemoryFile) GetReader(ctx context.Context) (io.ReadCloser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, exists := f.store.data[f.info.ID]
	if !exists {
		return nil, os.ErrNotExist
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *MemoryFile) ConcatUploads(ctx context.Context, uploads []handler.Upload) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store.mu.Lock()
	defer f.store.mu.Unlock()

	var result []byte
	for _, u := range uploads {
		if memFile, ok := u.(*MemoryFile); ok {
			data, exists := f.store.data[memFile.info.ID]
			if !exists {
				return fmt.Errorf("upload %s not found", memFile.info.ID)
			}
			result = append(result, data...)
		}
	}

	// Append concatenated data to the final upload's data
	existing := f.store.data[f.info.ID]
	finalData := append(existing, result...)
	f.store.data[f.info.ID] = finalData
	f.info.Offset = int64(len(finalData))

	if f.info.Offset == f.info.Size {
		f.isComplete = true
	}

	return nil
}

func (f *MemoryFile) ServeContent(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	reader, err := f.GetReader(ctx)
	if err != nil {
		return err
	}
	defer reader.Close()

	http.ServeContent(w, r, "", time.Time{}, reader.(io.ReadSeeker))
	return nil
}

func (f *MemoryFile) writeInfo() error {
	f.store.mu.Lock()
	defer f.store.mu.Unlock()

	data, err := json.Marshal(f.info)
	if err != nil {
		return err
	}
	f.store.metadata[f.info.ID] = data
	return nil
}

// MemoryStore implements handler.DataStore for in-memory storage
type MemoryStore struct {
	mu             sync.Mutex
	data           map[string][]byte
	metadata       map[string][]byte
	path           string
	DirModePerm    fs.FileMode
	FileModePerm   fs.FileMode
}

// New creates a new in-memory storage backend
func New(path string) *MemoryStore {
	return &MemoryStore{
		data:          make(map[string][]byte),
		metadata:      make(map[string][]byte),
		path:          path,
		DirModePerm:   os.FileMode(0755),
		FileModePerm:  os.FileMode(0644),
	}
}

// generateID generates a unique upload ID
func generateID() string {
	// Simple ID generation - tus.uid can also be used but we avoid external imports here
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// UseIn sets this store as the core data store in the passed composer
func (store *MemoryStore) UseIn(composer *handler.StoreComposer) {
	composer.UseCore(store)
	composer.UseTerminater(store)
	composer.UseConcater(store)
	composer.UseLengthDeferrer(store)
	composer.UseContentServer(store)
}

// NewUpload creates a new upload
func (store *MemoryStore) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	if info.ID == "" {
		info.ID = generateID()
	}

	binPath := store.path + "/" + info.ID
	infoPath := binPath + ".info"
	
	// Set up Storage map like filestore does
	info.Storage = map[string]string{
		"Type":      "memory",
		"Path":      binPath,
		"InfoPath":  infoPath,
	}

	// Create empty binary data (like filestore's empty file)
	if info.Size > 0 {
		store.data[info.ID] = make([]byte, 0, info.Size)
	} else {
		store.data[info.ID] = make([]byte, 0)
	}

	upload := &MemoryFile{
		info:    info,
		binPath: binPath,
		store:   store,
	}

	err := upload.writeInfo()
	if err != nil {
		delete(store.data, info.ID)
		return nil, err
	}

	return upload, nil
}

// GetUpload retrieves an existing upload
func (store *MemoryStore) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	metadata, exists := store.metadata[id]
	if !exists {
		return nil, handler.ErrNotFound
	}

	var info handler.FileInfo
	err := json.Unmarshal(metadata, &info)
	if err != nil {
		return nil, err
	}

	// Get actual data size and set offset, just like filestore
	// filestore does: stat, err := os.Stat(binPath); info.Offset = stat.Size()
	data, dataExists := store.data[id]
	if !dataExists {
		return nil, handler.ErrNotFound
	}
	info.Offset = int64(len(data))

	upload := &MemoryFile{
		info:       info,
		binPath:    store.path + "/" + id,
		store:      store,
		isComplete: info.Offset == info.Size,
	}

	return upload, nil
}

// GetReader returns an io.ReadCloser for the upload content
func (store *MemoryStore) GetReader(hCtx context.Context, info handler.FileInfo) (io.ReadCloser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	data, exists := store.data[info.ID]
	if !exists {
		return nil, handler.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (store *MemoryStore) AsTerminatableUpload(upload handler.Upload) handler.TerminatableUpload {
	return upload.(*MemoryFile)
}

func (store *MemoryStore) AsLengthDeclarableUpload(upload handler.Upload) handler.LengthDeclarableUpload {
	return upload.(*MemoryFile)
}

func (store *MemoryStore) AsConcatableUpload(upload handler.Upload) handler.ConcatableUpload {
	return upload.(*MemoryFile)
}

func (store *MemoryStore) AsServableUpload(upload handler.Upload) handler.ServableUpload {
	return upload.(*MemoryFile)
}

// GetUploadsInfo returns info for all uploads
func (store *MemoryStore) GetUploadsInfo(ctx context.Context) ([]handler.FileInfo, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var infos []handler.FileInfo
	for _, metadata := range store.metadata {
		var info handler.FileInfo
		if err := json.Unmarshal(metadata, &info); err != nil {
			continue
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// AssembleUploads returns a ReadCloser with concatenated upload data
func (store *MemoryStore) AssembleUploads(ctx context.Context, partials []string) (io.ReadCloser, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var readers []io.Reader
	for _, id := range partials {
		data, exists := store.data[id]
		if !exists {
			return nil, os.ErrNotExist
		}
		readers = append(readers, bytes.NewReader(data))
	}
	return io.NopCloser(io.MultiReader(readers...)), nil
}

// GetData returns the stored data for an upload (for testing)
func (store *MemoryStore) GetData(id string) ([]byte, bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	data, exists := store.data[id]
	return data, exists
}

// HasUpload checks if an upload exists
func (store *MemoryStore) HasUpload(id string) bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, exists := store.data[id]
	return exists
}

// GetUploadCount returns the total number of uploads stored
func (store *MemoryStore) GetUploadCount() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return len(store.data)
}

// ListUploads returns all upload IDs stored in the memory store
func (store *MemoryStore) ListUploads() []string {
	store.mu.Lock()
	defer store.mu.Unlock()

	ids := make([]string, 0, len(store.data))
	for id := range store.data {
		ids = append(ids, id)
	}
	return ids
}
