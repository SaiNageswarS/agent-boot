import logging
from temporalio import activity
from azure_storage import AzureStorage

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class IndexerActivities:
    def __init__(self, config: dict[str, str], azure_storage: AzureStorage):
        self._config = config
        self._azure_storage = azure_storage

    @activity.defn(name="convert_pdf_to_md")
    async def convert_pdf_to_md(self, tenant: str, pdf_file_name: str) -> str:
        """
        Convert a PDF file stored in Azure Blob Storage to Markdown format.
        
        Args:
            blob_name (str): The name of the blob in Azure Blob Storage.
        
        Returns:
            str: The path to the converted Markdown file.
        """
        import pymupdf4llm 

        blob_name = f"{tenant}/{pdf_file_name}"
        logging.info(f"Starting conversion of {blob_name} to Markdown")
        
        # Download the PDF file from Azure Blob Storage
        pdf_file_path = self._azure_storage.download_file(blob_name)
        
        # Convert the PDF to Markdown (placeholder for actual conversion logic)
        md_text = pymupdf4llm.to_markdown(pdf_file_path)
        
        # upload to Azure Blob Storage
        md_file_path = self._azure_storage.upload_bytes(blob_name.replace('.pdf', '.md'), md_text.encode('utf-8'))
        
        # Here you would implement the actual conversion logic
        # For now, we just simulate it by renaming the file

        logging.info(f"Converted {pdf_file_path} to {md_file_path}")
        
        return md_file_path

