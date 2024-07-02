"""
Uses llama 8b model to find inputs required in context of a query.
Uses few shot learning and chain-of-thought prompting.

Provides few examples (few shot) for model to understand output for input.
Uses chain of thought by providing interpretation as reasoning for model output.
"""

import re
import json

from pydantic import BaseModel
from llama_index.llms.groq import Groq
from llama_index.core.llms import ChatMessage


class IntentResult(BaseModel):
    context: list[str]
    interpretation: str


system_prompt = """
**Contextualizer Prompt**

You are a context expert, and your task is to identify the necessary context required to answer a user's query. 
You will be provided with a query, and you need to generate a response that includes the following:

1. **Context**: A list of specific contexts required to answer the query. This can include user-specific 
    information (e.g., user_profile, scores), web results (e.g., test syllabus, healthy lifestyle), 
    or other relevant information.
2. **Interpretation**: A brief explanation of why the specified contexts are required to answer the query.

Please respond in the following format:
{ "context": [...list of contexts...], "interpretation": "...brief explanation..." }

Here are some examples of how this prompt can be used:

**Example 1:**
**Query:** What are my scores?
**Response:**
{ "context": ["user_profile", "scores"], "interpretation": "To convey personalized user scores, scores of 
    user and profile are required" }
    
**Example 2:**
**Query:** How should I prepare for UPSC?
**Response:**
{ "context": ["user_profile", "scores", "web result: UPSC syllabus"], "interpretation": "To help user prepare 
    for test, current user scores are required and syllabus for test is required from web." }
    
**Example 3:**
**Query:** Which river flows from east to west?
**Response:**
{ "context": ["web result: river flowing from east to west"], "interpretation": "Factual question which is 
    required from the web." }
    
**Example 4:**
**Query:** Civil disobidience movement in India
**Response:**
{"context": ["web result: Civil disobidience movement in India"], "interpretation": "Factual question which 
    is required from the web." }

**Example 5:**
**Query:** Civil disobidience movement in India
**Response:**
{"context": ["web result: Civil disobidience movement in India"], "interpretation": "Factual question which 
    is required from the web." }
"""


def few_shot_intent_classification(query: str) -> IntentResult:
    llm = Groq(model="llama3-8b-8192")

    messages = [
        ChatMessage(role="system", content=system_prompt),
        ChatMessage(role="user", content=query),
    ]

    resp = llm.chat(messages)
    result = resp.message.content
    result_json = extract_json(result)

    return IntentResult(**result_json)


def extract_json(text: str):
    match = re.search(r'\{.*\}', text, re.DOTALL)
    if match:
        json_str = match.group(0)
        try:
            # Parse the JSON response
            result_json = json.loads(json_str)
            return result_json
        except json.JSONDecodeError:
            raise ValueError("The response from the LLM is not valid JSON.")
    else:
        raise ValueError("The response from the LLM does not contain valid JSON.")