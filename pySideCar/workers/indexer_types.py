from dataclasses import dataclass
from typing import List, Dict
from typing import Optional
from enum import Enum


@dataclass
class Chunk:
    chunkId: str
    sectionPath: str
    sectionIndex: int                       # Index of the section in the path
    phiRemoved: bool                        # true if PHI is removed, false otherwise
    sourceUri: str                          # e.g., "file://path/to/file.pdf"
    body: str                               # The actual content of the chunk
    tags: Optional[List[str]] = None        # Optional tags for the chunk
    abbrevations: Optional[Dict[str, str]] = None  # Optional abbreviations mapping


class Enhancement(str, Enum):
    NONE = "none"
    MEDICAL_ENTITIES = "medical_entities"


def parse_chunk_from_file(file_path: str) -> Chunk:
    """
    Parse a JSON file containing chunk definitions into a list of Chunk objects.
    
    Args:
        file_path (str): Path to the JSON file containing chunk definitions.
        
    Returns:
        Chunk: Chunk objects parsed from the file.
    """
    import json
    with open(file_path, "r") as f:
        section_dict = json.load(f)
        return Chunk(**section_dict)

