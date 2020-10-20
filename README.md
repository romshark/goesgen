<a href="https://github.com/romshark/goesgen/actions?query=workflow%3ACI">
    <img src="https://github.com/romshark/goesgen/workflows/CI/badge.svg" alt="GitHub Actions: CI">
</a>
<a href="https://coveralls.io/github/romshark/goesgen">
    <img src="https://coveralls.io/repos/github/romshark/goesgen/badge.svg" alt="Coverage Status" />
</a>
<a href="https://goreportcard.com/report/github.com/romshark/goesgen">
    <img src="https://goreportcard.com/badge/github.com/romshark/goesgen" alt="GoReportCard">
</a>
<a href="https://pkg.go.dev/github.com/romshark/goesgen">
    <img src="https://godoc.org/github.com/romshark/goesgen?status.svg" alt="GoDoc">
</a>

# goesgen
Go code generator that reduces the complexity of developing [Event-Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html) and [CQRS](https://martinfowler.com/bliki/CQRS.html) based microservice systems with optional [strong consistency guarantees](https://en.wikipedia.org/wiki/Strong_consistency) through [optimistic concurrency control](https://en.wikipedia.org/wiki/Optimistic_concurrency_control).

[goesgen](https://github.com/romshark/goesgen) generates boilerplate code from a declarative YAML schema file reducing the likelihood of errors (Services not subscribing to all necessary events, unused events, improper implementation of optimistic concurrency control on top of a transactional event log etc.)
