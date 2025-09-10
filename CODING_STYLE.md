# Coding Style Guide

This document outlines the coding style and conventions for the Go-Proxy project. Consistency is key, so please adhere to these guidelines to maintain a clean and readable codebase.

## Go Language Conventions

*   **Formatting:** All Go code should be formatted with `gofmt`. This is the standard for all Go projects and ensures a consistent format.
*   **Naming:**
    *   Use `camelCase` for local variables and function parameters.
    *   Use `PascalCase` for exported identifiers (functions, types, variables, etc.).
    *   Keep names short and descriptive.
    *   Avoid unnecessary stutter (e.g., `user.UserName` should be `user.Name`).
*   **Comments:**
    *   Write comments to explain the *why*, not the *what*.
    *   Use `//` for single-line comments and `/* */` for multi-line comments.
    *   Exported functions and types should have a doc comment explaining their purpose.
*   **Error Handling:**
    *   Errors are values. Handle them explicitly.
    *   Use `fmt.Errorf` to add context to errors.
    *   Don't discard errors with `_` unless you have a good reason.
*   **Imports:**
    *   Group imports into standard library, third-party, and internal packages.
    *   Use `goimports` to automatically format and group imports.

## Project-Specific Conventions

*   **Logging:** Use the `logger` package for all logging. Avoid using `fmt.Println` or `log.Println` directly.
*   **Configuration:** All configuration should be handled by the `config` package. Do not use hardcoded values.
*   **API:** The `api` package is responsible for all API-related functionality. Keep the API handlers clean and focused on handling HTTP requests.
*   **Storage:** The `storage` package is responsible for all interactions with Redis. Do not access Redis directly from other packages.

## Example

Here is an example of a well-formatted Go function that follows the project's coding style:

```go
// GetUser retrieves a user from the database by their ID.
func GetUser(id int) (*User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("invalid user ID: %d", id)
    }

    // ... code to retrieve user from the database ...

    return user, nil
}
```
