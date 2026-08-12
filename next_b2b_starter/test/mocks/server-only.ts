// Test shim for the "server-only" package. Next.js uses it to guard server
// modules from client bundles; vitest resolves the alias so server-side auth
// modules (server-constants, actions, etc.) can be imported in unit tests.
export {};
