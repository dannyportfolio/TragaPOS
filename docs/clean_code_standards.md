# Clean Code Standards & Engineering Policies

## Core Principles

1. **Self-Documenting Code Over Comments:**
   - Production source files and business logic contain zero inline comments (`//`).
   - Function names, variable names, and type definitions must clearly declare their intent and role.
   - Code that seems complex must be simplified through small, focused helper functions rather than explained with explanatory comments.

2. **Single Responsibility Principle (SRP):**
   - Each package and struct has a single reason to change.
   - HTTP delivery handlers are split per domain aggregate rather than clustered in a monolithic struct.
   - Data access resides solely in repositories; business rules reside in use cases; contracts reside in domain models.

3. **Dependency Inversion Principle (DIP):**
   - Higher-level modules (use cases) do not depend on lower-level modules (repositories or drivers). Both depend on abstractions defined in `internal/domain`.
   - Transaction boundaries are managed via the `domain.TransactionManager` interface, completely decoupling use cases from database drivers.

4. **Explicit Domain Error Handling:**
   - Prefer sentinel errors (`domain.ErrItemNotFound`, `domain.ErrTicketExpired`) and error wrapping (`%w`) over ad-hoc strings.
   - Guard clauses are used early to reduce nested `if` statements.

5. **Externalized Documentation:**
   - All architecture overviews, API contracts, domain models, and deployment configurations are maintained in the dedicated `docs/` directory.
