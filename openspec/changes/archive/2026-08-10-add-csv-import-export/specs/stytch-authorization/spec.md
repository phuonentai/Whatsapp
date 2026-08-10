## ADDED Requirements

### Requirement: RBAC export action for bulk download

The Stytch RBAC policy SHALL define an `export` action on the `contact`, `deal`, and `activity` resources, and the `admin` and `manager` roles SHALL be granted it, so that bulk data download is a distinct, explicitly-granted privilege rather than an implied consequence of `view`. The Go backend SHALL mirror the action in its fallback role-permission maps (`rbac.go`/`roles.go`) for development and mock-auth parity. The Redis-cached RBAC policy (`rbacPolicyCacheKey`) SHALL be versioned when the action is introduced so the new action takes effect without waiting for the cache TTL to expire.

#### Scenario: Export action exists in the Stytch policy

- **WHEN** the Stytch RBAC policy is fetched
- **THEN** the `contact`, `deal`, and `activity` resources SHALL list `export` among their actions
- **AND** the `admin` and `manager` roles SHALL be granted `contact:export`, `deal:export`, and `activity:export`

#### Scenario: Wildcard roles resolve export

- **WHEN** a role is granted `contact:*` in the policy
- **THEN** the expanded permissions SHALL include `contact:export` because `export` is a declared action of the `contact` resource

#### Scenario: Policy cache reflects the new action after rollout

- **WHEN** the policy cache key is versioned at rollout
- **THEN** the next fetch of the policy SHALL resolve the new `export` action without waiting for the prior cache TTL
