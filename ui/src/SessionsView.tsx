import { useEffect, useMemo, useState } from "react";
import {
    deleteSession,
    getAllSessions,
    updateSessionDuration,
    type Session,
} from "./api";
const DAYS_PER_PAGE = 7;

function formatDate(dateText: string): string {
    const date = new Date(dateText);

    if (Number.isNaN(date.getTime())) {
        return dateText;
    }

    const weekday = date.toLocaleDateString("en-GB", {
        weekday: "short",
    });

    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();

    return `${weekday} ${day}.${month}.${year}`;
}

function formatTime(dateText: string): string {
    const date = new Date(dateText);

    if (Number.isNaN(date.getTime())) {
        return "";
    }

    return date.toLocaleTimeString("en-GB", {
        hour: "2-digit",
        minute: "2-digit",
    });
}

function normalizeDurationInput(value: string): string {
    return value.trim().replaceAll(" ", "");
}

function getTimeText(dateText: string): string {
    const date = new Date(dateText);

    if (!Number.isNaN(date.getTime())) {
        return date.toLocaleTimeString("en-GB", {
            hour: "2-digit",
            minute: "2-digit",
        });
    }

    const timeMatch = dateText.match(/(\d{1,2}):(\d{2})/);

    if (timeMatch) {
        return `${timeMatch[1].padStart(2, "0")}:${timeMatch[2]}`;
    }

    return "";
}

export function SessionsView() {
    const [sessions, setSessions] = useState<Session[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const [editingSessionId, setEditingSessionId] = useState<string | null>(null);
    const [durationInput, setDurationInput] = useState("");
    const [saving, setSaving] = useState(false);

    const [visibleDayCount, setVisibleDayCount] = useState(DAYS_PER_PAGE);

    const sortedSessions = useMemo(() => {
        return [...sessions].sort((a, b) => {
            return new Date(b.start).getTime() - new Date(a.start).getTime();
        });
    }, [sessions]);


    const groupedSessions = useMemo(() => {
        const groups = new Map<string, Session[]>();

        for (const session of sortedSessions) {
            const day = session.focus_day;

            if (!groups.has(day)) {
                groups.set(day, []);
            }

            groups.get(day)!.push(session);
        }

        return Array.from(groups.entries()).map(([day, sessions]) => ({
            day,
            sessions,
        }));
    }, [sortedSessions]);

    const visibleGroups = groupedSessions.slice(0, visibleDayCount);
    const hasMoreGroups = visibleDayCount < groupedSessions.length;

    async function loadSessions() {
        setLoading(true);
        setError("");

        try {
            const result = await getAllSessions();
            setSessions(result);
            setVisibleDayCount(DAYS_PER_PAGE);
        } catch (error) {
            setError(error instanceof Error ? error.message : "Failed to load sessions");
        } finally {
            setLoading(false);
        }
    }

    useEffect(() => {
        loadSessions();
    }, []);

    function startEditing(session: Session) {
        setEditingSessionId(session.id);
        setDurationInput(session.duration_human.replaceAll(" ", ""));
        setError("");
    }

    function cancelEditing() {
        setEditingSessionId(null);
        setDurationInput("");
        setError("");
    }

    async function saveDuration(sessionId: string) {
        const normalizedDuration = normalizeDurationInput(durationInput);

        if (!normalizedDuration) {
            setError("Duration is required");
            return;
        }

        setSaving(true);
        setError("");

        try {
            await updateSessionDuration(sessionId, normalizedDuration);
            setEditingSessionId(null);
            setDurationInput("");
            await loadSessions();
        } catch (error) {
            setError(error instanceof Error ? error.message : "Failed to update session");
        } finally {
            setSaving(false);
        }
    }

    async function handleDelete(session: Session) {
        const confirmed = window.confirm(
            `Delete this session?\n\n${formatDate(session.start)}\n${formatTime(
                session.start,
            )} — ${formatTime(session.end)}\n${session.duration_human}`,
        );

        if (!confirmed) {
            return;
        }

        setSaving(true);
        setError("");

        try {
            await deleteSession(session.id);
            await loadSessions();
        } catch (error) {
            setError(error instanceof Error ? error.message : "Failed to delete session");
        } finally {
            setSaving(false);
        }
    }

    return (
        <section className="card sessionsCard">
            <p className="eyebrow">Management</p>
            <h1>Sessions</h1>

            {loading && <p className="emptyState">Loading sessions...</p>}

            {error && <p className="error">{error}</p>}

            {!loading && !error && sortedSessions.length === 0 && (
                <p className="emptyState">No saved focus sessions yet.</p>
            )}

            {!loading && groupedSessions.length > 0 && (
                <>
                    <div className="sessionsList">
                        {visibleGroups.map((group) => (
                            <section className="sessionDayGroup" key={group.day}>
                                <div className="sessionDayHeader">
                                    <h2>Day: {group.day}</h2>

                                    <span className="sessionDayCount">
                                        {group.sessions.length} session
                                        {group.sessions.length === 1 ? "" : "s"}
                                    </span>
                                </div>

                                <div className="sessionRows">
                                    {group.sessions.map((session) => {
                                        const isEditing = editingSessionId === session.id;

                                        return (
                                            <article className="sessionItem" key={session.id}>
                                                <div className="sessionRange">
                                                    {getTimeText(session.start)} - {getTimeText(session.end)}
                                                </div>

                                                {isEditing ? (
                                                    <div className="sessionEditRow">
                                                        <input
                                                            className="durationInput"
                                                            value={durationInput}
                                                            onChange={(event) =>
                                                                setDurationInput(event.target.value)
                                                            }
                                                            placeholder="5m or 1h30m"
                                                            disabled={saving}
                                                        />

                                                        <button
                                                            className="sessionMiniButton"
                                                            onClick={() => saveDuration(session.id)}
                                                            disabled={saving}
                                                        >
                                                            Save
                                                        </button>

                                                        <button
                                                            className="sessionMiniButton sessionCancelButton"
                                                            onClick={cancelEditing}
                                                            disabled={saving}
                                                        >
                                                            Cancel
                                                        </button>
                                                    </div>
                                                ) : (
                                                    <div className="sessionDuration">
                                                        ⏱ {session.duration_human}
                                                    </div>
                                                )}

                                                <div className="sessionActions">
                                                    {!isEditing && (
                                                        <button
                                                            className="sessionMiniButton"
                                                            onClick={() => startEditing(session)}
                                                            disabled={saving}
                                                        >
                                                            Edit
                                                        </button>
                                                    )}

                                                    <button
                                                        className="sessionDeleteButton"
                                                        onClick={() => handleDelete(session)}
                                                        disabled={saving}
                                                        aria-label="Delete session"
                                                        title="Delete session"
                                                    >
                                                        ×
                                                    </button>
                                                </div>
                                            </article>
                                        );
                                    })}
                                </div>
                            </section>
                        ))}
                    </div>

                    {hasMoreGroups && (
                        <button
                            className="loadMoreButton"
                            onClick={() =>
                                setVisibleDayCount((current) => current + DAYS_PER_PAGE)
                            }
                        >
                            Load more
                        </button>
                    )}
                </>
            )}
        </section>
    );
}