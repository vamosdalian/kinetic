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

export async function apiClient<T>(
  input: RequestInfo | URL,
  init?: RequestInit
): Promise<T> {
  try {
    const response = await fetch(input, init);

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
