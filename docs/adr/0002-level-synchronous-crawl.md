# ADR 0002: Level-synchronous breadth-first crawl

## Status

Accepted

## Context

With a worker pool, the order in which pages finish depends on timing. A
naive shared queue makes the set of pages crawled under `--max-pages`
depend on scheduling, so two runs against an unchanged site could produce
different reports and noisy baseline diffs.

## Decision

The crawler processes one depth level at a time. URLs discovered at depth
`d` are collected, de-duplicated, sorted and truncated to the remaining page
budget before depth `d+1` starts. Workers fetch a level concurrently, but
the set of URLs in each level is independent of timing.

## Consequences

- Reports are reproducible for a stable site regardless of `--concurrency`.
- Memory for one level is bounded by the frontier limit.
- A very wide level is fetched entirely before deeper pages, which is the
  expected breadth-first behaviour for an audit.
