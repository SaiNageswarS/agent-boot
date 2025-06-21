import asyncio
import os
import logging
import configparser
from dotenv import load_dotenv

from temporalio.client import Client
from temporalio.worker import Worker

from azure_storage import AzureStorage
from workers.indexer_activities import IndexerActivities

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


async def main():
    load_dotenv(override=True)
    run_mode = os.getenv("ENV", "dev").lower()

    config = configparser.ConfigParser()
    config.read("config.ini")
    env_config = dict(config[run_mode])

    azure_storage = AzureStorage(env_config)
    activities = IndexerActivities(env_config, azure_storage)

    temporal_host = env_config["temporal_host_port"]

    client = await Client.connect(temporal_host)

    worker = Worker(
        client,
        task_queue=env_config["temporal_py_task_queue"],
        activities=[activities.convert_pdf_to_md, activities.window_section_chunks],
    )

    logger.info("🚀 Starting Temporal Worker...")
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
