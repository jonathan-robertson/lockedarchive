# File Cabinet
This project is meant for storing/archiving family information in a safe way: Documents, Tax forms, scans of important licenses, etc.

Due to the sensitive nature of this kind of information, encryption is paramount and will be applied before data is uploaded through the internet (even though https is being used).

And since quick access to valuable information is obviously important to everyone, I'm also planning on support for Tagging, Searching (not just names, but content inside documents as well), Date stamps (since 'last modified' doesn't reflect when my bill was due), and decrypting/viewing a document without storing it decrypted on the harddrive (no unencrypted files left behind).

Status | Feature
:---: | ---
in progress | Store documents with an object storage account you own (encrypted before upload)
planned | Set a date for your document (due date for bill, day receipt was received, etc.)
planned | View your documents without storing decrypted data to local HDD
researching | Tag documents (mark bill as 'paid', paperwork as 'submitted', etc.)
researching | Search document names **and contents** to find information without having to download file contents

Status | Object Storage
:---: | ---
in progress | Amazon S3
researching | Backblaze B2
researching | Google Cloud Storage
maybe one day? | Rackspace Cloud Files

# Documentation Licensing
The content of this document and all other documents within this repository are licensed under the Creative Commons Attribution 3.0 License.
