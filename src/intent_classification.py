"""
Uses llama 8b model to find inputs required in context of a query.
Uses few shot learning and chain-of-thought prompting.

Provides few examples (few shot) for model to understand output for input.
Uses chain of thought by providing interpretation as reasoning for model output.
"""

from pydantic import BaseModel
from llama_index.llms.groq import Groq
from llama_index.core.llms import ChatMessage


class IntentResult(BaseModel):
    query: str
    contextRequired: list[str]
    interpretation: str


class IntentExample(BaseModel):
    query: str
    contextRequired: list[str]
    interpretation: str


system_prompt_template = """
You are a context expert, and your task is to identify the necessary context required to answer a user's query. 
You will be provided with a query, please respond in the following format, replacing placeholders with actual content.

query placeholder | list of context required separated by [SEP] placeholder | interpretation placeholder.

**Important Guidelines**:

1. Be specific and concise in your responses.
2. Identify the minimum necessary context required to answer the query.
3. Consider both user-specific information (e.g., user_profile, scores) and external knowledge (e.g., web results) that may be needed to provide a relevant answer.
4. Provide a clear and concise interpretation of why the specified context is required to answer the query.
5. Use [SEP] separator for list of context required.

Here are some examples of how this prompt can be used:
{Examples}
"""

context_example_template = """
**Query:** {query}
**Response:**
{query} | {contextRequired} | {interpretation}
"""


def few_shot_intent_classification(query: str, examples: list[IntentExample]) -> IntentResult:
    llm = Groq(model="llama3-8b-8192")
    system_prompt = __generate_prompt__(examples)

    messages = [
        ChatMessage(role="system", content=system_prompt),
        ChatMessage(role="user", content=query),
    ]

    resp = llm.chat(messages)
    result_str = resp.message.content
    result = __extract_intent_result__(result_str)
    return result


def __generate_prompt__(examples: list[IntentExample]) -> str:
    examples_str = ""

    for example in examples:
        examples_instance = context_example_template.format(
            query=example.query,
            contextRequired="[SEP]".join(example.contextRequired),
            interpretation=example.interpretation)

        examples_str = examples_str + examples_instance

    result = system_prompt_template.format(Examples=examples_str)
    return result


def __extract_intent_result__(text: str) -> IntentResult:
    result_parts = text.split("|")

    if len(result_parts) != 3:
        raise ValueError("LLM response is malformed.")

    context_required = result_parts[1].split("[SEP]")
    context_required = [x.strip() for x in context_required]
    result = IntentResult(query=result_parts[0], contextRequired=context_required, interpretation=result_parts[2])
    return result
