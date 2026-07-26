import { useEffect, useMemo, useState } from "react";
import { getAllSessions, type Session } from "./api";

type DaySummary = {
    date: Date;
    dateKey: string;
    totalSeconds: number;
    activities: ActivityDaySummary[];
};

type ActivityDaySummary = {
    activityID: string | null;
    title: string;
    totalSeconds: number;
};

type WeekGroup = {
    weekKey: string;
    weekNumber: number;
    days: DaySummary[];
    totalSeconds: number;
};

type WeekSort = "newest" | "least-time" | "most-time";

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
    const activitiesByDate = new Map<
        string,
        Map<string, ActivityDaySummary>
    >();

    for (const session of sessions) {
        const currentTotal = totalsByDate.get(session.focus_day) ?? 0;

        totalsByDate.set(
            session.focus_day,
            currentTotal + session.duration_seconds,
        );

        let dayActivities = activitiesByDate.get(session.focus_day);

        if (!dayActivities) {
            dayActivities = new Map();
            activitiesByDate.set(session.focus_day, dayActivities);
        }

        const activityKey = session.activity?.id ?? "no-activity";
        const currentActivity = dayActivities.get(activityKey);

        if (currentActivity) {
            currentActivity.totalSeconds += session.duration_seconds;
        } else {
            dayActivities.set(activityKey, {
                activityID: session.activity?.id ?? null,
                title: session.activity?.title ?? "No activity",
                totalSeconds: session.duration_seconds,
            });
        }
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
            const activities = Array.from(
                activitiesByDate.get(dateKey)?.values() ?? [],
            );

            days.push({
                date,
                dateKey,
                totalSeconds: totalsByDate.get(dateKey) ?? 0,
                activities,
            });
        }

        const totalSeconds = days.reduce((total, day) => {
            return total + day.totalSeconds;
        }, 0);

        weeks.push({
            weekKey: formatDateKey(currentMonday),
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

function getWeekActivitySummary(
    week: WeekGroup,
): ActivityDaySummary[] {
    const totals = new Map<string, ActivityDaySummary>();

    for (const day of week.days) {
        for (const activity of day.activities) {
            const activityKey =
                activity.activityID ?? "no-activity";

            const currentActivity = totals.get(activityKey);

            if (currentActivity) {
                currentActivity.totalSeconds += activity.totalSeconds;
                continue;
            }

            totals.set(activityKey, {
                activityID: activity.activityID,
                title: activity.title,
                totalSeconds: activity.totalSeconds,
            });
        }
    }

    return Array.from(totals.values()).sort(
        (a, b) => b.totalSeconds - a.totalSeconds,
    );
}

function formatWeekRange(week: WeekGroup): string {
    const firstDay = week.days[0];
    const lastDay = week.days[week.days.length - 1];

    if (!firstDay || !lastDay) {
        return "";
    }

    const formatDate = (date: Date): string => {
        return date.toLocaleDateString("en-GB", {
            day: "2-digit",
            month: "short",
            year: "numeric",
        });
    };

    return `${formatDate(firstDay.date)} — ${formatDate(lastDay.date)}`;
}

export function QuickHistoryView() {
    const [sessions, setSessions] = useState<Session[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [weekSort, setWeekSort] = useState<WeekSort>("newest");
    const [selectedDayKey, setSelectedDayKey] = useState<string | null>(null);
    const [selectedWeekKey, setSelectedWeekKey] =
        useState<string | null>(null);

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

    function getSortedWeeks(weeks: WeekGroup[]): WeekGroup[] {
        return [...weeks].sort((a, b) => {
            if (weekSort === "least-time") {
                return a.totalSeconds - b.totalSeconds;
            }

            if (weekSort === "most-time") {
                return b.totalSeconds - a.totalSeconds;
            }

            return a.weekNumber - b.weekNumber;
        });
    }

    return (
        <section className="card historyCard">
            <p className="eyebrow">History</p>
            <h1>Quick History</h1>

            <div className="historyToolbar">
                <div className="historySort">
                    <span>Sort weeks</span>

                    <select
                        value={weekSort}
                        onChange={(event) =>
                            setWeekSort(event.target.value as WeekSort)
                        }
                    >
                        <option value="newest">Newest first</option>
                        <option value="most-time">Most focused time</option>
                        <option value="least-time">Least focused time</option>
                    </select>
                </div>
            </div>

            {loading && <p className="emptyState">Loading history...</p>}

            {error && <p className="error">{error}</p>}

            {!loading && !error && sessions.length === 0 && (
                <p className="emptyState">No saved focus sessions yet.</p>
            )}

            {!loading && !error && weeks.length > 0 && (
                <div className="quickWeeks">
                    {getSortedWeeks(
                        weeks.filter((week) => week.totalSeconds > 0),
                    ).map((week) => {
                        const selectedDay = week.days.find(
                            (day) => day.dateKey === selectedDayKey,
                        );

                        const isWeekStatsOpen = selectedWeekKey === week.weekKey;

                        const activeDays = week.days.filter(
                            (day) => day.totalSeconds > 0,
                        ).length;

                        const averagePerDay = Math.round(
                            week.totalSeconds / 7,
                        );

                        const averagePerActiveDay =
                            activeDays > 0
                                ? Math.round(week.totalSeconds / activeDays)
                                : 0;

                        const weekActivities = getWeekActivitySummary(week);

                        return (
                            <section className="quickWeek" key={week.weekKey}>
                                <div className="quickWeekHeader">
                                    <h2>Week {week.weekNumber}</h2>

                                    <button
                                        type="button"
                                        className={
                                            selectedWeekKey === week.weekKey
                                                ? "weekStatsButton weekStatsButtonActive"
                                                : "weekStatsButton"
                                        }
                                        onClick={() => {
                                            setSelectedDayKey(null);

                                            setSelectedWeekKey((currentWeekKey) =>
                                                currentWeekKey === week.weekKey
                                                    ? null
                                                    : week.weekKey,
                                            );
                                        }}
                                        aria-expanded={selectedWeekKey === week.weekKey}
                                        aria-label={`Toggle statistics for week ${week.weekNumber}`}
                                        title="Week statistics"
                                    >
                                        ≡
                                    </button>
                                </div>

                                <div className="quickDaysGrid">
                                    {week.days
                                        .filter((day) => day.totalSeconds > 0)
                                        .map((day) => (
                                            <button
                                                type="button"
                                                className={
                                                    selectedDayKey === day.dateKey
                                                        ? "quickDayCell quickDayCellSelected"
                                                        : "quickDayCell"
                                                }
                                                key={day.dateKey}
                                                onClick={() => {
                                                    setSelectedWeekKey(null);

                                                    setSelectedDayKey((currentDayKey) =>
                                                        currentDayKey === day.dateKey
                                                            ? null
                                                            : day.dateKey,
                                                    );
                                                }}
                                            >
                                                <span className="quickDayDate">
                                                    {formatDayTitle(day.date)}
                                                </span>

                                                <strong className="quickDayTotal">
                                                    ⏱ {formatDuration(day.totalSeconds)}
                                                </strong>
                                            </button>
                                        ))}
                                </div>
                                {isWeekStatsOpen && (
                                    <div className="quickWeekDetails">
                                        <div className="quickWeekDetailsHeader">
                                            <div className="quickWeekDetailsTitle">
                                                <strong>Week {week.weekNumber}</strong>
                                                <span className="quickWeekRange">
                                                    {formatWeekRange(week)}
                                                </span>
                                            </div>
                                        </div>

                                        <div className="quickWeekTotal">
                                            <span>Total focused</span>

                                            <strong>
                                                {formatDuration(week.totalSeconds)}
                                            </strong>
                                        </div>

                                        <div className="quickWeekMetrics">
                                            <div className="quickWeekMetric">
                                                <span>Active days</span>
                                                <strong>{activeDays}</strong>
                                            </div>

                                            <div className="quickWeekMetric">
                                                <span>Daily average</span>
                                                <strong>{formatDuration(averagePerDay)}</strong>
                                            </div>

                                            <div className="quickWeekMetric">
                                                <span>Average active day</span>
                                                <strong>
                                                    {formatDuration(averagePerActiveDay)}
                                                </strong>
                                            </div>
                                        </div>

                                        <div className="quickWeekActivityList">
                                            {weekActivities.map((activity) => {
                                                const percentage =
                                                    week.totalSeconds > 0
                                                        ? Math.round(
                                                            (activity.totalSeconds / week.totalSeconds) * 100,
                                                        )
                                                        : 0;

                                                return (
                                                    <div
                                                        className="quickWeekActivityRow"
                                                        key={activity.activityID ?? "no-activity"}
                                                    >
                                                        <span
                                                            className="quickWeekActivityTitle"
                                                            title={activity.title}
                                                        >
                                                            {activity.title}
                                                        </span>

                                                        <span className="quickWeekActivityPercent">
                                                            {percentage}%
                                                        </span>

                                                        <strong>
                                                            {formatDuration(activity.totalSeconds)}
                                                        </strong>
                                                    </div>
                                                );
                                            })}
                                        </div>
                                    </div>
                                )}

                                {selectedDay && (
                                    <div className="quickDayDetails">
                                        <div className="quickDayDetailsHeader">
                                            <strong>{formatDayTitle(selectedDay.date)}</strong>

                                            <span>
                                                Total: {formatDuration(selectedDay.totalSeconds)}
                                            </span>
                                        </div>

                                        <div className="quickDayActivityList">
                                            {[...selectedDay.activities]
                                                .sort((a, b) => b.totalSeconds - a.totalSeconds)
                                                .map((activity) => (
                                                    <div
                                                        className="quickDayActivityRow"
                                                        key={activity.activityID ?? "no-activity"}
                                                    >
                                                        <span
                                                            className="quickDayActivityTitle"
                                                            title={activity.title}
                                                        >
                                                            {activity.title}
                                                        </span>

                                                        <strong className="quickDayActivityDuration">
                                                            {formatDuration(activity.totalSeconds)}
                                                        </strong>
                                                    </div>
                                                ))}
                                        </div>
                                    </div>
                                )}
                            </section>
                        );
                    })}
                </div>
            )}
        </section>
    );
}