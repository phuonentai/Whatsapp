# Getting Started
 
 ## 1. Quick Setup
 
 1. **Clone & Install**
    ```bash
    git clone <repository-url>
    cd next_b2b_starter
    ./setup.sh
    ```
    *(This script sets up Docker, database migrations, and .env files)*
 
 2. **Run Backend**
    ```bash
    cd go-b2b-starter
    make dev
    ```
 
 3. **Run Frontend**
    ```bash
    cd next_b2b_starter
    pnpm dev
    ```
 
 Visit `http://localhost:3000`.
 
 ## 2. Project Structure
 
 - **`app/`**: Next.js App Router (Pages & APIs).
 - **`components/`**: Shared UI components.
 - **`lib/`**: Business logic, API clients, and hooks.
 - **`middleware.ts`**: Auth protection.
 
 ## 3. Common Issues
 
 - **Environment Variables**: Ensure `app.env` is loaded by the backend.
 - **Ports**: Backend runs on `8080`, Frontend on `3000`.
 
 ## Next Steps
 
 👉 **Learn about**: [Authentication](./02-authentication.md)
