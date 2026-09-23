# Burp Suite Extension

This extension connects Burp Suite traffic to Pyntra so the platform can review requests, responses, and related testing context.

## Build
Use the provided Gradle or Maven helper scripts in this directory to produce the JAR under `dist/`.

## Usage Notes
1. Load the generated JAR in Burp Suite.
2. Configure the Pyntra endpoint and authentication settings.
3. Validate the connection with a small test request before broader usage.
