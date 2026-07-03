import { useEffect, useMemo, useState } from "react";
import { getSessions, type Session } from "./api";

type DayCell = {
    date: Date;
    dateKey: string;
    isCurrentMonth: boolean;
    totalSeconds: number;
};

function formatDateKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function formatShortDate(date: Date): string {
    const day = date.toLocaleDateString("en-GB", { weekday: "short" });
    const dateText = date.toLocaleDateString("en-GB", {
        day: "numeric",
        month: "numeric",
    });

    return `${day} ${dateText}`;
}

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

function getMonthRange(date: Date): { from: string; to: string } {
    const year = date.getFullYear();
    const month = date.getMonth();

    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);

    return {
        from: formatDateKey(firstDay),
        to: formatDateKey(lastDay),
    };
}

function getWeekTotalSeconds(week: DayCell[]): number {
    return week.reduce((total, day) => total + day.totalSeconds, 0);
}

function getCalendarWeeks(monthDate: Date, sessions: Session[]): DayCell[][] {
    const year = monthDate.getFullYear();
    const month = monthDate.getMonth();

    const firstDayOfMonth = new Date(year, month, 1);
    const lastDayOfMonth = new Date(year, month + 1, 0);

    const start = new Date(firstDayOfMonth);
    const startDay = start.getDay(); // Sunday = 0, Monday = 1
    const daysToMonday = startDay === 0 ? 6 : startDay - 1;
    start.setDate(start.getDate() - daysToMonday);

    const end = new Date(lastDayOfMonth);
    const endDay = end.getDay();
    const daysToSunday = endDay === 0 ? 0 : 7 - endDay;
    end.setDate(end.getDate() + daysToSunday);

    const totalsByDate = new Map<string, number>();

    for (const session of sessions) {
        const dateKey = formatDateKey(new Date(session.start));
        const currentTotal = totalsByDate.get(dateKey) ?? 0;

        totalsByDate.set(dateKey, currentTotal + session.duration_seconds);
    }

    const weeks: DayCell[][] = [];
    let currentWeek: DayCell[] = [];

    const current = new Date(start);

    while (current <= end) {
        const dateKey = formatDateKey(current);

        currentWeek.push({
            date: new Date(current),
            dateKey,
            isCurrentMonth: current.getMonth() === month,
            totalSeconds: totalsByDate.get(dateKey) ?? 0,
        });

        if (currentWeek.length === 7) {
            weeks.push(currentWeek);
            currentWeek = [];
        }

        current.setDate(current.getDate() + 1);
    }

    return weeks;
}

export function HistoryView() {
    const [monthDate] = useState(() => new Date());
    const [sessions, setSessions] = useState<Session[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const monthTitle = monthDate.toLocaleDateString("en-GB", {
        month: "long",
        year: "numeric",
    });

    const weeks = useMemo(() => {
        return getCalendarWeeks(monthDate, sessions);
    }, [monthDate, sessions]);

    useEffect(() => {
        async function loadHistory() {
            setLoading(true);
            setError("");

            try {
                const { from, to } = getMonthRange(monthDate);
                const result = await getSessions(from, to);

                setSessions(result);
            } catch (error) {
                setError(error instanceof Error ? error.message : "Failed to load history");
            } finally {
                setLoading(false);
            }
        }

        loadHistory();
    }, [monthDate]);

    return (
        <section className="card historyCard">
            <p className="eyebrow">History</p>
            <h1>{monthTitle}</h1>

            {loading && <p className="emptyState">Loading history...</p>}
            {error && <p className="error">{error}</p>}

            {!loading && !error && (
                <div className="weeks">
                    {weeks.map((week, index) => {
                        const weekTotalSeconds = getWeekTotalSeconds(week);

                        return (
                            <section className="week" key={index}>
                                <h2>
                                    Week {index + 1}{" "}
                                    <span className="weekTotal">({formatDuration(weekTotalSeconds)})</span>
                                </h2>

                                <div className="daysGrid">
                                    {week.map((day) => (
                                        <div
                                            className={day.isCurrentMonth ? "dayCell" : "dayCell mutedDay"}
                                            key={day.dateKey}
                                        >
                                            <span className="dayDate">{formatShortDate(day.date)}</span>

                                            <span className="dayDivider" aria-hidden="true" />

                                            <span className="dayTotal">
                                                <span className="timeIcon" aria-hidden="true">⏱</span>
                                                <strong>{formatDuration(day.totalSeconds)}</strong>
                                            </span>
                                        </div>
                                    ))}
                                </div>
                            </section>
                        );
                    })}
                </div>
            )}
        </section>
    );
}