---
default: patch
---

# Remove the duplicated meta export service and refresh the client spec

Removes the meta export service (CID DAG and Sia object endpoints) that
is served by the dedicated meta service and provided by the portal-sdk
module. Refreshes the swagger spec and regenerated client against the
deployed ipfs endpoint definition, including 422 validation responses.
