# D16. Every text a person reads follows simple-english

Status: Decided.

Ken asked for this on 2026-09-27. Load the `simple-english` skill before you
write any text that a person reads: a document, a code comment, an error
message or a commit message. Follow it for that text.

Its rules include the Go conventions for error messages in D7. It adds more:
short sentences, the active voice, `can`, `will` and `must` in place of
`should` and `may`, no semicolons, no em dashes, no contractions, the
condition before the command, and one word for one meaning.
