"""
Flask interface file to call classifiers over http network.
"""

from flask import Flask, request, jsonify
from pydantic import BaseModel, Field, field_validator, ValidationError
import json
from src.intent_classification import IntentExample, few_shot_intent_classification
from src.personalized_response_generator import PersonalizedResponseGenerator
from src.knowledge_base import KnowledgeBase

app = Flask(__name__)


@app.route('/infer/intent', methods=['POST'])
def infer_intent():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = IntentRequest(**data)
        result = few_shot_intent_classification(validated_data.query, validated_data.examples)

        return jsonify(result.dict()), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


@app.route('/infer/personalizedResponse', methods=['POST'])
def personalized_response():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = PersonalizationRequest(**data)
        result = PersonalizedResponseGenerator().generate(
            query=validated_data.query,
            context=validated_data.context,
            kb_query=validated_data.kb_query,
            threshold=validated_data.threshold)

        return jsonify({'response': result}), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


@app.route('/kb/indexQuery', methods=['POST'])
def add_query():
    try:
        data = request.json
        if not data:
            return jsonify({'error': 'No data provided'}), 400

        # Validate incoming data using Pydantic
        validated_data = IndexQueryRequest(**data)
        kb = KnowledgeBase()
        result = kb.index_query(
            qid=validated_data.qid,
            query=validated_data.query,
            metadata=validated_data.metadata)

        return jsonify({'isSuccess': result}), 200
    except ValidationError as e:
        return jsonify({'error': e.errors()}), 422


class IntentRequest(BaseModel):
    query: str
    examples: list[IntentExample]


class PersonalizationRequest(BaseModel):
    query: str
    context: dict[str, str]
    kb_query: str
    threshold: float

    @field_validator('context', mode='before')
    def parse_metadata(cls, value):
        if isinstance(value, str):
            return json.loads(value)
        return value


class IndexQueryRequest(BaseModel):
    qid: str
    query: str
    metadata: dict[str, str]

    @field_validator('metadata', mode='before')
    def parse_metadata(cls, value):
        if isinstance(value, str):
            return json.loads(value)
        return value


if __name__ == '__main__':
    app.run(debug=True)
