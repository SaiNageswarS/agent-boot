from dataclasses import dataclass
from typing import List, Dict
from typing import Optional
import orjson


@dataclass
class Chunk:
    chunkId: str
    title: str
    sectionPath: str
    sectionIndex: int     # Index of the section in the path
    sourceUri: str      
    sentences: List[str]  # The actual content of the chunk
    prevChunkId: str 
    nextChunkId: str
    sectionId: str      # Stable hash for the section
    windowIndex: int    # 0-based window order within section
    tags: Optional[List[str]] = None  # Optional tags for the chunk
    abbrevations: Optional[Dict[str, str]] = None  # Optional abbreviations mapping

    def to_json_bytes(self) -> bytes:
        return orjson.dumps(
            self,
            option=orjson.OPT_SERIALIZE_DATACLASS,
        ) 


def parse_section_chunk_file(file_path: str) -> Chunk:
    """
    Parse a JSON file containing chunk definitions into a list of Chunk objects.

    Args:
        file_path (str): Path to the JSON file containing chunk definitions.

    Returns:
        Chunk: Chunk objects parsed from the file.
    """
    import json

    with open(file_path, "r", encoding="utf-8") as f:
        section_dict = json.load(f)
        return Chunk(**section_dict)
