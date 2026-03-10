// Package upload provides CAR file generation utilities with memory-efficient two-pass
// processing for IPFS content.
//
// This package implements memory-constrained CAR (Content Addressable aRchive) file
// generation using UnixFS for content addressing. The two-pass approach allows building
// large directory trees without keeping all blocks in memory:
//
// Pass 1: Walk the filesystem and build a summary with metadata (CIDs, sizes, tree structure)
// Pass 2: Write the CARv1 file, regenerating blocks from the filesystem on demand
//
// This approach enables processing directory trees larger than available memory by using
// an LRU blockstore with configurable memory limits.
//
// Core Components
//
//   - CARBuilder: Two-pass CAR generator with summary-based block regeneration
//   - LRUBlockstore: Memory-efficient blockstore with LRU eviction
//   - UnixFSNodeGenerator: Creates UnixFS nodes from readers
//
// Basic Usage
//
// Generate a CAR file from a filesystem:
//
//	ctx := context.Background()
//	filesystem := os.DirFS("/path/to/content")
//
//	// Create CAR with 100MB memory limit
//	carFile, err := os.Create("output.car")
//	if err != nil {
//	    return err
//	}
//	defer carFile.Close()
//
//	rootCID, err := StreamCAR(ctx, filesystem, carFile, 100*units.MiB, true)
//	if err != nil {
//	    return err
//	}
//
// fmt.Printf("CAR file created with root CID: %s\n", rootCID)
//
// Memory Management
//
// The LRU blockstore automatically evicts older blocks when memory limits are
// exceeded. Default memory limit is 100MB but can be customized:
//
// StreamCARWithSize(ctx, filesystem, w, 200*units.MiB, wrapInDir)
//
// Advanced Usage
//
// For finer control over the CAR generation process, use CARBuilder directly:
//
//	bs, dagService := NewDAGServiceWithMemoryLimit(DefaultMemoryLimit)
//	generator := NewUnixFSNodeGenerator(
//	    WithUnixFSNodeDAGService(dagService),
//	    WithUnixFSNodeBlockstore(bs),
//	)
//
//	builder := NewCARBuilder(bs, dagService, generator)
//
//	// Pass 1: Build summary
//	summary, err := builder.BuildSummary(ctx, filesystem, wrapInDir)
//	if err != nil {
//	    return err
//	}
//
//	// Pass 2: Write CAR
//	if err := builder.WriteCAR(ctx, carFile); err != nil {
//	    return err
//	}
package upload
