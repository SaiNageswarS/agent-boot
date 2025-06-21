import logging
import tempfile
from pathlib import Path
from azure.identity import DefaultAzureCredential
from azure.storage.blob import BlobServiceClient

# Set up logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class AzureStorage:
    def __init__(self, config: dict[str, str]):
        self._config = config
        self._blob_client = None

    @property
    def blob_client(self) -> BlobServiceClient:
        if self._blob_client is None:
            account_name = self._config.get("azure_storage_account")
            if not account_name:
                raise ValueError("Missing 'azure_storage_account' in config")

            credential = DefaultAzureCredential()
            account_url = f"https://{account_name}.blob.core.windows.net"
            try:
                self._blob_client = BlobServiceClient(
                    account_url=account_url, credential=credential
                )
            except Exception as e:
                logger.exception("Failed to create Azure Blob client")
                raise RuntimeError("Blob client initialization failed") from e

        return self._blob_client

    def download_file(self, container_name: str, blob_name: str) -> str:
        file_name = Path(blob_name).name
        temp_dir = tempfile.mkdtemp(prefix="azure_download_")
        tmp_file_path = Path(temp_dir).joinpath(file_name)

        try:
            blob = self.blob_client.get_blob_client(
                container=container_name, blob=blob_name
            )
            with open(tmp_file_path, "wb") as f:
                stream = blob.download_blob()
                f.write(stream.readall())
        except Exception as e:
            logger.exception(f"Failed to download blob '{blob_name}'")
            raise RuntimeError(f"Download failed for blob '{blob_name}'") from e

        logger.info("File downloaded successfully: %s", tmp_file_path)
        return str(tmp_file_path)

    def upload_bytes(self, container_name: str, blob_name: str, data: bytes) -> str:
        try:
            blob = self.blob_client.get_blob_client(
                container=container_name, blob=blob_name
            )
            blob.upload_blob(data, overwrite=True)
        except Exception as e:
            logger.exception(f"Failed to upload data to blob '{blob_name}'")
            raise RuntimeError(f"Upload failed for blob '{blob_name}'") from e

        blob_url = blob.url
        logger.info("Data uploaded successfully: %s", blob_url)
        return blob_url
