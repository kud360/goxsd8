// Package codeccensus is the census of codec over the W3C suite that `go tool
// decodecensus` and `go tool encodecensus` report: [Walk] reads the
// population, assembles and generates each case's schema, resolves the type
// each case decodes into, and builds and runs the drivers whose summaries
// each command merges and prints. Each command owns its report; this package
// owns the population, the assembly and the drivers, and the [Caveats] they
// put on every figure; package assess, beneath it, owns how a driver
// classifies one instance or one round trip.
//
// # Population
//
// The population is the instance lane's: every instanceTest entry
// conformance.Catalog reads from the suite at the root and does not mark
// withheld. A case's schema is its test group's <schemaDocument> list,
// assembled by assess.Assemble: the first document through loader.Dir over
// its own directory, then each further one its closure did not read. The
// instance lane assembles more widely than that, as a caveat the report prints
// states. A group's schema generates when it assembles and codegen.Generate
// emits it, as one target, a package of its own in a temporary module, with no
// error; a case of a group declaring other than one schemaTest has no schema
// and does not generate.
//
// # Drivers
//
// codec.Decode and codec.Append each take a compiled T, so the census compiles
// what it generated: the temporary module, at the path driverModule names,
// requires this module by a replace directive on the module root `go env
// GOMOD` names from the working directory, so that its generated main packages
// may import package assess. Each batch of up to batchSize packages builds one
// driver with `go build`, which calls assess.Drive, for decodecensus, or
// assess.DriveEncode, for encodecensus (entryOf), with one assess.CodecFor
// Codec per type the batch's cases decode into, bound to one
// codec.WithBranches per interface of that type's package, or for a
// PrecisionDecimal's Value interface one codec.WithPrecisionDecimalArms, and
// one codec.WithBranches per instantiation of codegen's generic Nillable the
// package's fields or decoded document elements use, which the census declares
// in a file of its own beside the generated one (writeNillableOptions), so
// that the census supplies every package's branch and arm sets
// (sealedInterfaces). A command calling Walk therefore runs inside this
// module's tree, with the go command on PATH and the module cache holding this
// module's requirements, as GOFLAGS=-mod=mod builds offline from it.
//
// A case's T is the type the generated file declares, by its header comment
// and its name, for the top-level element declaration with {abstract} false
// that the instance's document element names: ownType states the match; for
// a declaration codegen gives a nilled root's value Nillable over that type,
// T is that Nillable (nillableFamily.root). A case whose document element is
// not well-formed, names no such declaration, or names one ownType, or
// nillableFamily.root, matches no single type for, decodes into a
// placeholder struct instead, and [Population] counts those cases.
//
// An absent suite is a supported mode, not a failure: the submodule is
// absent in a fresh container (#659), so Walk says so, naming the command
// that initializes it, and reports the suite absent rather than an error.
package codeccensus
