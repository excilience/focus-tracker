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
    activity: ActivitySummary | null;
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
    activity: ActivitySummary | null;
};

export type GoalResponse = {
    focus_day: string;
    focused_seconds: number;
    focused_human: string;
    goal_seconds: number;
    goal_human: string;
    remaining_seconds: number;
    remaining_human: string;
    percent: number;
    is_completed: boolean;
}

export type SettingsResponse = {
    day_start_hour: number;
};

export function getSettings(): Promise<SettingsResponse> {
    return request<SettingsResponse>("/settings");
}

export function updateSettings(dayStartHour: number): Promise<SettingsResponse> {
    return request<SettingsResponse>("/settings", {
        method: "PATCH",
        body: JSON.stringify({
            day_start_hour: dayStartHour,
        }),
    });
}

export function getGoal(): Promise<GoalResponse> {
    return request<GoalResponse>("/goal");
}


export function updateGoal(dailyGoal: string): Promise<GoalResponse> {
    return request<GoalResponse>("/goal", {
        method: "PATCH",
        body: JSON.stringify({
            daily_goal: dailyGoal.replaceAll(" ", ""),
        })
    });
}

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

export function startSession(activityID?: string): Promise<MessageResponse> {
    return request<MessageResponse>("/sessions/start", {
        method: "POST",
        body: JSON.stringify({
            activity_id: activityID ?? null,
        }),
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

//activity
export type ActivitySummary = {
    id: string;
    title: string;
};

export type Activity = {
    id: string;
    title: string;
    is_archived: boolean;
    created_at: string;
};

export type ActivityStats = {
    activity: ActivitySummary | null;
    duration_seconds: number;
    duration_human: string;
};

export function getActivities(): Promise<Activity[]> {
    return request<Activity[]>("/activities");
}

export function getActivityStats(): Promise<ActivityStats[]> {
    return request<ActivityStats[]>("/activities/stats");
}

export function createActivity(title: string): Promise<Activity> {
    return request<Activity>("/activities", {
        method: "POST",
        body: JSON.stringify({
            title,
        }),
    });
}

export type UpdateActivityRequest = {
    title?: string;
    is_archived?: boolean;
};

export function updateActivity(activityID: string, updates: UpdateActivityRequest): Promise<Activity> {
    return request<Activity>(`/activities/${activityID}`, {
        method: "PATCH",
        body: JSON.stringify(updates)
    });
}