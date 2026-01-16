---
applyTo: "**"
---

# Kinetic Project Instructions

This project is named Kinetic. It is a workflow orchestration tool designed to help users automate and manage complex workflows efficiently. The project includes both a backend API server and a frontend web application.

## Common Guidelines

1. Annotations should be in English and should not be made unless necessary

## Backend

Language: Go
requirement:
  1. Use logrus for logging. 
  2. Always add unit tests for new features or bug fixes.

## Frontend

Language: TypeScript with React and Vite
requirement:
  1. Use the provided `apiClient` utility for all API interactions instead of the native `fetch` function.
  2. Use sahdcn/ui component library for UI components and styling.
  3. Avoid using React memo too early unless the user explicitly requests them.
  4. If you need to run commands such as npm in the frontend, you need to cd to the web directory first.