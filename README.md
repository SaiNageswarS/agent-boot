# agent-boot

Advanced RAG (Retrieval-Augmented Generation) agent for building AI applications. It has primarily two components - SearchCore and MCP (Model Context Protocol) Agent.

SearchCore has worker to index documents and a search engine to retrieve relevant documents based on the query. It supports various data sources like web pages, PDFs, and more.

MCP agent interacts with Claude Desktop to extract queries and intent from user input, and then uses SearchCore to retrieve relevant documents. It can also generate responses based on the retrieved documents.

This particular repo is designed for biomedical search and retrieval, leveraging advanced LLMs (Large Language Models) and vector databases to provide accurate and context-aware results.

# Approach

## 1 Ingestion & Knowledge-Enrichment

Stage
1.1 Parsing & chunking	• PDF→structured XML (Grobid)
• Split into 128-token “evidence windows” with 32-token overlap.	Keeps each chunk within model context.
1.2 Biomedical annotation	• UMLS / MeSH concept tagging (MetaMap or SciSpacy)
• Section ID (intro, methods, results, conclusions).	Later lets us filter by intent (e.g., “root-cause” → methods/results).
1.3 Objective extraction	Ask GPT-4o: “Give one line describing this chunk’s objective (cause / treatment / prognosis / prevalence / mechanism / misc).”	Supplies an objective sentence for intent matching.
1.4 Multi-vector embedding	For every chunk we store four vectors:
1. Universal – OpenAI text-embedding-3-large (1 536-d).

2. Domain – SapBERT-Sentence (768-d). 
GitHub
PMC

3. Long-doc – SciNCL document embedding (768-d). 
arXiv

4. Objective – BiomedBERT fine-tuned with SimCSE on the extracted objective (768-d).	Gives four complementary similarity signals.

All vectors, tags, and metadata are persisted in a multi-index store:

scss
Copy
Edit
Qdrant collection
 ├── universal_vec  (HNSW, cosine)
 ├── sapbert_vec    (HNSW, cosine)
 ├── longdoc_vec    (IVF-PQ, cos)
 └── objective_vec  (HNSW, cosine)
ElasticSearch index (BM25 + SPLADE)
Neo4j graph         (UMLS concept relations)

## 2 Query & Intent Understanding
LLM pre-processor (GPT-4o 128k context)

Generates ≥5 paraphrased queries (HyDE style).

Classifies intent (cause, diagnosis, treatment, prognosis, mechanism, trial_match, …).

Extracts entities & temporal filters.

Intent embedding – same BiomedBERT-Objective encoder as above so intent lives in the same vector space.

Discrete intent tag – used for hard filters (e.g., only “cause” sections).

## 3 Three-Tier Retrieval Cascade
Tier	Candidates	Technique
T1: High-recall fan-out	50 000	• BM25+SPLADE on Elastic (lexical)
• Universal & SapBERT ANN search
• UMLS graph expansion
T2: Dense fusion	3 000	Reciprocal Rank Fusion (RRF) of the four vector channels + BM25 scores.
T3: Intent-aware filtering	500	Drop chunks whose objective tag ≠ intent; if still >500, keep highest intent-objective cosine.

## 4 Heavyweight Re-Ranking & Fusion
Layer	Model	Notes
RR-1	MedCPT Cross-Encoder (PubMed-3B, listwise) 
arXiv
Batch-scores all 500 on A100 cluster (≈0.3 s).
RR-2	BioLinkBERT cross-encoder (large, MedNLI-tuned) 
PMC
Ranks top-200 from MedCPT; adds domain nuance.
RR-3	GPT-4o listwise reranker	Prompt: “Rank these passages for intent <X>, emphasise new causal evidence.” Processes 100 candidates in one 128k call.
Diversity pass	MMR with a concept-coverage term (UMLS nodes)	Prevents near-duplicates.

The final Top-N (≈20) passages flow forward with full provenance (doc ID, section, score breakdown).

## 5 Answer Synthesis & Evidence Graph
Chain-of-Thought LLM (GPT-4o w/ code interpreter):

Reason step-by-step over the top-20 passages.

Generates a diagnostic hypothesis tree with probability estimates.

Emits inline citations to passage IDs.

Dynamic evidence graph (Neo4j) built from cited UMLS concepts ± their relations → lets the doctor expand a node and see supporting literature.

Counter-factual check – second GPT-4o call: “List opposing evidence or alternative explanations not covered above.”

## 6 Offline Model Distillation (optional)
Because cost is no object, you still distil:

Train a ColBERT-v2 retriever on the cross-encoder judgments for nightly re-index, shrinking tiers T1+T2 latency by 10× for interactive demo. 
Medium
arXiv

Fine-tune BiomedBERT SimCSE with cross-encoder listwise distillation for an even better objective embedding. 
arXiv

## 7 Evaluation Loop
Metric	Dataset	Target (no-cost setting)
nDCG@10	BioASQ-12 factoid & list	≥ 0.80
Recall@50	PubMed QA custom set	≥ 0.95
Physician relevance voting	300 real cases	≥ 85 % “Highly useful”
Hallucination rate	Adversarial symptomatic prompts	≤ 2 %

Any slice that slips triggers automatic model surgery (e.g., more MedCPT epochs).

Why this is the “best imaginable” pipeline
Four complementary vector spaces ensure nothing relevant is missed.

Intent threads from query understanding → retrieval filters → ranking so every step is purpose-aware.

Cross-encoder → LLM rerank stack reaches state-of-the-art accuracy on present leaderboards, unconstrained by cost or GPU time.

Evidence graph + counter-factual check delivers explainability and reduces hallucination risk—critical for biomedical use.

Distillation stages let you later trade cost for speed without losing quality.

With unlimited budget and latency tolerance, this is as close as 2025 technology can get to an “oracle” biomedical search engine.


