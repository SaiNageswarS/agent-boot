import logging
from dataclasses import asdict
from temporalio import activity

from azure_storage import AzureStorage

from workers.indexer_types import parse_section_chunk_file
from workers.window_chunker import WindowChunker

# Set up logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class IndexerActivities:
    def __init__(self, config: dict[str, str], azure_storage: AzureStorage):
        self._config = config
        self._azure_storage = azure_storage

        self.window_chunker = WindowChunker()

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

        logging.info(f"Starting conversion of {pdf_file_name} to Markdown")

        # Download the PDF file from Azure Blob Storage
        pdf_file_path = self._azure_storage.download_file(tenant, pdf_file_name)

        # Convert the PDF to Markdown (placeholder for actual conversion logic)
        md_text = pymupdf4llm.to_markdown(pdf_file_path)

        # upload to Azure Blob Storage
        md_file_name = pdf_file_name.replace(".pdf", ".md")
        self._azure_storage.upload_bytes(tenant, md_file_name, md_text.encode("utf-8"))

        # Here you would implement the actual conversion logic
        # For now, we just simulate it by renaming the file

        logging.info(f"Converted {pdf_file_name} to {md_file_name}")

        return md_file_name

    @activity.defn(name="window_section_chunks")
    async def window_section_chunks(
        self,
        tenant: str,
        md_section_json_urls: list[str],
        windows_output_path: str,
    ) -> list[str]:
        """
        Process Markdown sections into windowed chunks.

        Args:
            tenant (str): The tenant identifier.
            md_section_json_url (str): JSON URL of a single Markdown section.
            windows_output_path (str): Output path for the windowed chunks.
        Returns:
            list[str]: Storage blob path of windows.
        """
        logging.info(
            f"Processing {len(md_section_json_urls)} Markdown sections for tenant {tenant}"
        )

        # Convert JSON string to list of sections
        import json

        result = []

        for idx, md_section_json_url in enumerate(md_section_json_urls):
            md_section_json_file = self._azure_storage.download_file(
                tenant, md_section_json_url
            )
            md_section = parse_section_chunk_file(md_section_json_file)

            # Process the sections into windowed chunks
            window_chunks = self.window_chunker.chunk_windows(md_section)

            for window_chunk in window_chunks:
                window_chunk_json = json.dumps(asdict(window_chunk))
                blob_path = f"{windows_output_path}/{window_chunk.chunkId}.chunk.json"
                self._azure_storage.upload_bytes(
                    tenant, blob_path, window_chunk_json.encode("utf-8")
                )

                result.append(blob_path)

            activity.heartbeat({"progress": f"{idx+1}/{len(md_section_json_urls)}"})

        return result
