import { clearStoredAuthToken, getStoredAuthToken, redirectToLogin } from "@/lib/auth"

export interface APIResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: string;
  };
  meta?: {
    page: number;
    pageSize: number;
    total: number;
    totalPages: number;
  };
}

export interface APIClientInit extends RequestInit {
  skipAuthRedirect?: boolean
  skipAuthToken?: boolean
}

function buildRequestInit(init?: APIClientInit): RequestInit | undefined {
  const token = init?.skipAuthToken ? "" : getStoredAuthToken()
  if (!init && !token) {
    return undefined
  }

  const headers = new Headers(init?.headers)
  if (token) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  return {
    ...init,
    headers,
  }
}

function handleUnauthorized(init?: APIClientInit) {
  clearStoredAuthToken()
  if (!init?.skipAuthRedirect) {
    redirectToLogin()
  }
}

export async function apiClient<T>(
  input: RequestInfo | URL,
  init?: APIClientInit
): Promise<T> {
  try {
    const response = await fetch(input, buildRequestInit(init));

    if (!response.ok) {
        // Try to parse error details if available
        let errorMsg = `HTTP error! status: ${response.status}`;
        try {
            const errorBody = await response.json();
             if (errorBody && errorBody.error && errorBody.error.message) {
                 errorMsg = errorBody.error.message;
             }
        } catch {
            // Ignore JSON parse error for error response
        }
      if (response.status === 401) {
        handleUnauthorized(init)
      }
      console.error(errorMsg);
      throw new Error(errorMsg);
    }

    const json: APIResponse<T> = await response.json();

    if (!json.success) {
      const message = json.error?.message || "Unknown API error";
      console.error("API Error:", message, json.error);
      throw new Error(message);
    }

    return json.data;
  } catch (error) {
    console.error("Fetch error:", error);
    throw error;
  }
}

export async function apiClientFull<T>(
  input: RequestInfo | URL,
  init?: APIClientInit
): Promise<APIResponse<T>> {
  try {
    const response = await fetch(input, buildRequestInit(init));

    if (!response.ok) {
      // Try to parse error details if available
      let errorMsg = `HTTP error! status: ${response.status}`;
      try {
        const errorBody = await response.json();
        if (errorBody && errorBody.error && errorBody.error.message) {
          errorMsg = errorBody.error.message;
        }
      } catch {
        // Ignore JSON parse error for error response
      }
      if (response.status === 401) {
        handleUnauthorized(init)
      }
      console.error(errorMsg);
      throw new Error(errorMsg);
    }

    const json: APIResponse<T> = await response.json();

    if (!json.success) {
      const message = json.error?.message || "Unknown API error";
      console.error("API Error:", message, json.error);
      throw new Error(message);
    }

    return json;
  } catch (error) {
    console.error("Fetch error:", error);
    throw error;
  }
}

export const apiClientPublic = apiClient
