# VaultedPages [![Apache-2.0 License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/puddingfactory/vaultedpages/blob/master/LICENSE.md)

This project is meant for storing/archiving family information in a safe way: Documents, Tax forms, scans of important licenses, etc.

Due to the sensitive nature of this kind of information, encryption is paramount and will be applied before data is uploaded through the internet (though https is also being used).

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
considering | Backblaze B2
considering | Google Cloud Storage
considering | Rackspace Cloud Files

## Documentation Licensing

The content of this document and all other documents within this repository are licensed under the Creative Commons Attribution 3.0 License.
