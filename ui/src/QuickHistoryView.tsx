import { useEffect, useMemo, useState } from "react";
import { getAllSessions, type Session } from "./api";

type DaySummary = {
    date: Date;
    dateKey: string;
    totalSeconds: number;
};

type WeekGroup = {
    weekNumber: number;
    days: DaySummary[];
    totalSeconds: number;
};

function formatDuration(totalSeconds: number): string {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);

    if (hours > 0 && minutes > 0) {
        return `${hours}h ${minutes}m`;
    }

    if (hours > 0) {
        return `${hours}h`;
    }

    if (minutes > 0) {
        return `${minutes}m`;
    }

    return "0m";
}

function formatDateKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function formatDayTitle(date: Date): string {
    const weekdays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

    const weekday = weekdays[date.getDay()];
    const day = String(date.getDate()).padStart(2, "0");
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const year = date.getFullYear();

    return `${weekday} ${day}/${month}/${year}`;
}
function getMonday(date: Date): Date {
    const result = new Date(date);
    const day = result.getDay(); // Sunday = 0, Monday = 1
    const diff = day === 0 ? -6 : 1 - day;

    result.setDate(result.getDate() + diff);
    result.setHours(0, 0, 0, 0);

    return result;
}

function addDays(date: Date, days: number): Date {
    const result = new Date(date);
    result.setDate(result.getDate() + days);
    return result;
}

function parseDateKey(dateKey: string): Date {
    const [year, month, day] = dateKey.split("-").map(Number);
    return new Date(year, month - 1, day);
}

function groupSessionsByCalendarWeeks(sessions: Session[]): WeekGroup[] {
    if (sessions.length === 0) {
        return [];
    }

    const totalsByDate = new Map<string, number>();

    for (const session of sessions) {
        const currentTotal = totalsByDate.get(session.focus_day) ?? 0;
        totalsByDate.set(session.focus_day, currentTotal + session.duration_seconds);
    }

    const sortedFocusDays = Array.from(totalsByDate.keys()).sort();

    const firstFocusDate = parseDateKey(sortedFocusDays[0]);
    const lastFocusDate = parseDateKey(sortedFocusDays[sortedFocusDays.length - 1]);

    const firstMonday = getMonday(firstFocusDate);
    const lastMonday = getMonday(lastFocusDate);

    const weeks: WeekGroup[] = [];
    let currentMonday = new Date(firstMonday);

    while (currentMonday <= lastMonday) {
        const days: DaySummary[] = [];

        for (let dayIndex = 0; dayIndex < 7; dayIndex++) {
            const date = addDays(currentMonday, dayIndex);
            const dateKey = formatDateKey(date);

            days.push({
                date,
                dateKey,
                totalSeconds: totalsByDate.get(dateKey) ?? 0,
            });
        }

        const totalSeconds = days.reduce((total, day) => {
            return total + day.totalSeconds;
        }, 0);

        weeks.push({
            weekNumber: weeks.length + 1,
            days,
            totalSeconds,
        });

        currentMonday = addDays(currentMonday, 7);
    }

    return weeks.reverse().map((week, index) => ({
        ...week,
        weekNumber: index + 1,
    }));
}

export function QuickHistoryView() {
    const [sessions, setSessions] = useState<Session[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const weeks = useMemo(() => {
        return groupSessionsByCalendarWeeks(sessions);
    }, [sessions]);

    useEffect(() => {
        async function loadSessions() {
            setLoading(true);
            setError("");

            try {
                const result = await getAllSessions();
                setSessions(result);
            } catch (error) {
                setError(error instanceof Error ? error.message : "Failed to load sessions");
            } finally {
                setLoading(false);
            }
        }

        loadSessions();
    }, []);

    return (
        <section className="card historyCard">
            <p className="eyebrow">History</p>
            <h1>Quick History</h1>

            {loading && <p className="emptyState">Loading history...</p>}

            {error && <p className="error">{error}</p>}

            {!loading && !error && sessions.length === 0 && (
                <p className="emptyState">No saved focus sessions yet.</p>
            )}

            {!loading && !error && weeks.length > 0 && (
                <div className="quickWeeks">
                    {weeks
                        .filter((week) => week.totalSeconds > 0)
                        .map((week) => (
                            <section className="quickWeek" key={week.weekNumber}>
                                <h2>
                                    Week {week.weekNumber}{" "}
                                    <span className="weekTotal">
                                        ({formatDuration(week.totalSeconds)})
                                    </span>
                                </h2>

                                <div className="quickDaysGrid">
                                    {week.days
                                        .filter((day) => day.totalSeconds > 0)
                                        .map((day) => (
                                            <article className="quickDayCell" key={day.dateKey}>
                                                <span className="quickDayDate">
                                                    {formatDayTitle(day.date)}
                                                </span>

                                                <strong className="quickDayTotal">
                                                    ⏱ {formatDuration(day.totalSeconds)}
                                                </strong>
                                            </article>
                                        ))}
                                </div>
                            </section>
                        ))}
                </div>
            )}
        </section>
    );
}