# Object Store API in Golang

This is an S3 compliant object storage application.

We have a three layers of architecture:

1. Storage / BlobStore
2. ObjectStore / MetadataStore
3. Http / S3 Compliant API

To keep proper abstraction for each layer we have defined each contract independently and they interact with dependency injection.

Storage and BlobStore layer is where the actual object bytes live. This is currently implemented using FilesystemBlobStore and stores the data on local filesystem of the server.
The plan is to implement multiple BlobStores to support multiple architectures, including distributed storage nodes.

ObjectStore and MetadataStore are the layer where the objects themselves are managed. When an object is created, it gets a record in the metadata store. This keeps track of bucket, prefix, object id, and blob id values.
The blob id is how we interact and retrieve data from the blobstore system.

The Http / S3 Compliant API is the web service layer that can GET/PUT/DELETE Objects into the service. Currently S3 compliance is a goal and not quite a reality.
