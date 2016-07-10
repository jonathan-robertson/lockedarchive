# Purpose
I really just want to store my family's personal information online in a way that's easy to access without sacrificing any security.

# Design Goals

Concept | Description
:---: | ---
Portability | should work with Amazon S3, Rackspace Cloud Files, Backblaze B2, and others... even though designed only for S3 for now.
Data Security | end-to-end encryption required and doesn't rely on or take advantage of any provider-managed encryption. Data does not touch HDD in unencrypted format unless specifically requested by user (user clicks Download button). For Preview mode, cached encrypted data is decrypted on a per-request basis and disposed of once preview window is dismissed.
Metadata Security | metadata contains important information as well, so it must be encrypted. During 'cabinet load' phase, all metadata is fetched from storage provider, decrypted, and stored in memory (never to local HDD).
Accessibility | golang compiles to just about any language and I will at least be compiling for the big 3 (linux, darwin/mac, and windows).

# PDF Parsing Challenges
1. PDF read/parse support doesn't appear to be widely supported
- Storing text in metadata may not be a good option since "user-defined metadata is limited to 2 KB in size" according to S3.

# Documentation Licensing
The content of this document and all other documents within this repository are licensed under the Creative Commons Attribution 3.0 License.
