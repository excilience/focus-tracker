const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type ApiErrorResponse = {
    error: {
        code: string;
        message: string;
        details?: Record<string, unknown>;
    };
};

export type HealthResponse = {
    status: string;
};

export type ActiveSession = {
    start: string;
    last_resume: string;
    focused_seconds: number;
    focused_human: string;
    is_paused: boolean;
};

export type MessageResponse = {
    message: string;
};

export type Session = {
    id: string;
    start: string;
    end: string;
    focus_day: string;
    duration_seconds: number;
    duration_human: string;
};

export type DeleteSessionResponse = {
    message: string;
    session: Session;
};


export function getSessions(from: string, to: string): Promise<Session[]> {
    return request<Session[]>(`/sessions?from=${from}&to=${to}`);
}

export function getAllSessions(): Promise<Session[]> {
    return request<Session[]>("/sessions");
}

export type StopSessionResponse = {
    message: string;
    saved: boolean;
    session?: Session;
};

async function request<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${path}`, {
        ...options,
        headers: {
            "Content-Type": "application/json",
            ...options?.headers,
        },
    });

    if (!response.ok) {
        let apiError: ApiErrorResponse | null = null;

        try {
            apiError = await response.json();
        } catch {
            // response body is not JSON
        }

        throw new Error(apiError?.error?.message ?? `Request failed: ${response.status}`);
    }

    return response.json();
}

export function getHealth(): Promise<HealthResponse> {
    return request<HealthResponse>("/health");
}

export async function getActiveSession(): Promise<ActiveSession | null> {
    try {
        return await request<ActiveSession>("/sessions/active");
    } catch {
        return null;
    }
}

export function startSession(): Promise<MessageResponse> {
    return request<MessageResponse>("/sessions/start", {
        method: "POST",
    });
}

export function pauseSession(): Promise<MessageResponse> {
    return request<MessageResponse>("/sessions/pause", {
        method: "POST",
    });
}

export function resumeSession(): Promise<MessageResponse> {
    return request<MessageResponse>("/sessions/resume", {
        method: "POST",
    });
}

export function stopSession(): Promise<StopSessionResponse> {
    return request<StopSessionResponse>("/sessions/stop", {
        method: "POST",
    });
}

export function updateSessionDuration(sessionID: string, duration: string): Promise<Session> {
    return request<Session>(`/sessions/${sessionID}`, {
        method: "PATCH",
        body: JSON.stringify({
            duration: duration.replaceAll(" ", ""),
        }),
    });
}

export function deleteSession(sessionID: string): Promise<DeleteSessionResponse> {
    return request<DeleteSessionResponse>(`/sessions/${sessionID}`, {
        method: "DELETE",
    });
}