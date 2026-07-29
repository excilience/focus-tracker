import { useEffect, useMemo, useState } from "react";
import { getAllSessions, type Session } from "./api";


type MonthSort = "newest" | "least-time" | "most-time";

type DaySummary = {
    dateKey: string;
    date: Date;
    totalSeconds: number;
    sessionsCount: number;
};

type WeekSummary = {
    weekKey: string;
    weekNumber: number;
    totalSeconds: number;
    days: DaySummary[];
};

type MonthSummary = {
    monthKey: string;
    year: number;
    monthIndex: number;
    monthName: string;
    totalSeconds: number;
    daysWorked: number;
    weeks: WeekSummary[];
};

type YearSummary = {
    year: number;
    totalSeconds: number;
    monthsWorked: number;
    daysWorked: number;
    months: MonthSummary[];
};

type AllTimeStats = {
    startDate: Date | null;
    endDate: Date;
    totalSeconds: number;
    sessionsCount: number;
    activeDays: number;
    calendarDays: number;
    dailyAverageSeconds: number;
    averageActiveDaySeconds: number;
    activitiesCount: number;
};

type ActivityPeriodSummary = {
    activityID: string | null;
    title: string;
    totalSeconds: number;
    percentage: number;
};

type PeriodStats = {
    startDate: Date;
    endDate: Date;
    totalSeconds: number;
    sessionsCount: number;
    activeDays: number;
    calendarDays: number;
    dailyAverageSeconds: number;
    averageActiveDaySeconds: number;
    activities: ActivityPeriodSummary[];
};
type StatsPeriod =
    | "week"
    | "month"
    | "quarter"
    | "half-year"
    | "year"
    | "all-time"
    | "custom";

type DateRange = {
    startDate: Date;
    endDate: Date;
};

function getCalendarDaysCount(startDate: Date, endDate: Date): number {
    const start = new Date(startDate);
    const end = new Date(endDate);

    start.setHours(0, 0, 0, 0);
    end.setHours(0, 0, 0, 0);

    const differenceMilliseconds = end.getTime() - start.getTime();

    return Math.floor(differenceMilliseconds / 86400000) + 1;
}

function formatDuration(totalSeconds: number): string {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);

    if (hours > 0 && minutes > 0) return `${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h`;
    if (minutes > 0) return `${minutes}m`;
    return "0m";
}

function parseDateKey(dateKey: string): Date {
    const [year, month, day] = dateKey.split("-").map(Number);
    return new Date(year, month - 1, day);
}

function isDateInRange(
    date: Date,
    startDate: Date,
    endDate: Date,
): boolean {
    const time = date.getTime();

    return (
        time >= startDate.getTime() &&
        time <= endDate.getTime()
    );
}

function startOfDay(date: Date): Date {
    const result = new Date(date);
    result.setHours(0, 0, 0, 0);
    return result;
}

function endOfDay(date: Date): Date {
    const result = new Date(date);
    result.setHours(23, 59, 59, 999);
    return result;
}

function getMonthKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    return `${year}-${month}`;
}

function getMonthName(date: Date): string {
    return date.toLocaleDateString("en-GB", {
        month: "long",
    });
}

function getWeekNumber(date: Date): number {
    const target = new Date(date);
    target.setHours(0, 0, 0, 0);

    target.setDate(target.getDate() + 3 - ((target.getDay() + 6) % 7));

    const firstThursday = new Date(target.getFullYear(), 0, 4);
    firstThursday.setDate(
        firstThursday.getDate() + 3 - ((firstThursday.getDay() + 6) % 7),
    );

    return (
        1 +
        Math.round(
            ((target.getTime() - firstThursday.getTime()) / 86400000 - 3) / 7,
        )
    );
}

function formatDayLabel(date: Date): string {
    const weekday = date.toLocaleDateString("en-GB", {
        weekday: "short",
    });

    return `${weekday} ${date.getDate()}.`;
}

function buildGlobalHistory(sessions: Session[]): YearSummary[] {
    const daysByDate = new Map<string, DaySummary>();

    for (const session of sessions) {
        const dateKey = session.focus_day;
        const date = parseDateKey(dateKey);

        const currentDay = daysByDate.get(dateKey);

        if (currentDay) {
            currentDay.totalSeconds += session.duration_seconds;
            currentDay.sessionsCount += 1;
        } else {
            daysByDate.set(dateKey, {
                dateKey,
                date,
                totalSeconds: session.duration_seconds,
                sessionsCount: 1,
            });
        }
    }

    const yearsByYear = new Map<number, YearSummary>();

    const sortedDays = Array.from(daysByDate.values()).sort((a, b) => {
        return b.date.getTime() - a.date.getTime();
    });

    for (const day of sortedDays) {
        const year = day.date.getFullYear();
        const monthKey = getMonthKey(day.date);
        const monthIndex = day.date.getMonth();
        const weekNumber = getWeekNumber(day.date);
        const weekKey = `${year}-week-${weekNumber}`;

        let yearSummary = yearsByYear.get(year);

        if (!yearSummary) {
            yearSummary = {
                year,
                totalSeconds: 0,
                monthsWorked: 0,
                daysWorked: 0,
                months: [],
            };

            yearsByYear.set(year, yearSummary);
        }

        let monthSummary = yearSummary.months.find((month) => {
            return month.monthKey === monthKey;
        });

        if (!monthSummary) {
            monthSummary = {
                monthKey,
                year,
                monthIndex,
                monthName: getMonthName(day.date),
                totalSeconds: 0,
                daysWorked: 0,
                weeks: [],
            };

            yearSummary.months.push(monthSummary);
            yearSummary.monthsWorked += 1;
        }

        let weekSummary = monthSummary.weeks.find((week) => {
            return week.weekKey === weekKey;
        });

        if (!weekSummary) {
            weekSummary = {
                weekKey,
                weekNumber,
                totalSeconds: 0,
                days: [],
            };

            monthSummary.weeks.push(weekSummary);
        }

        yearSummary.totalSeconds += day.totalSeconds;
        yearSummary.daysWorked += 1;

        monthSummary.totalSeconds += day.totalSeconds;
        monthSummary.daysWorked += 1;

        weekSummary.totalSeconds += day.totalSeconds;
        weekSummary.days.push(day);
    }

    const years = Array.from(yearsByYear.values());

    for (const year of years) {
        year.months.sort((a, b) => b.monthIndex - a.monthIndex);

        for (const month of year.months) {
            month.weeks.sort((a, b) => b.weekNumber - a.weekNumber);

            for (const week of month.weeks) {
                week.days.sort((a, b) => a.date.getTime() - b.date.getTime());
            }
        }
    }

    return years.sort((a, b) => b.year - a.year);
}

function getStatsDateRange(
    period: StatsPeriod,
    sessions: Session[],
): DateRange {
    const endDate = startOfDay(new Date());
    const startDate = new Date(endDate);

    if (period === "week") {
        startDate.setDate(startDate.getDate() - 6);
    }

    if (period === "month") {
        startDate.setDate(startDate.getDate() - 29);
    }

    if (period === "quarter") {
        startDate.setMonth(startDate.getMonth() - 3);
        startDate.setDate(startDate.getDate() + 1);
    }

    if (period === "half-year") {
        startDate.setMonth(startDate.getMonth() - 6);
        startDate.setDate(startDate.getDate() + 1);
    }

    if (period === "year") {
        startDate.setFullYear(startDate.getFullYear() - 1);
        startDate.setDate(startDate.getDate() + 1);
    }

    if (period === "all-time") {
        if (sessions.length === 0) {
            return {
                startDate: endDate,
                endDate,
            };
        }

        let earliestDate = parseDateKey(
            sessions[0].focus_day,
        );

        for (const session of sessions) {
            const sessionDate = parseDateKey(
                session.focus_day,
            );

            if (
                sessionDate.getTime() <
                earliestDate.getTime()
            ) {
                earliestDate = sessionDate;
            }
        }

        return {
            startDate: earliestDate,
            endDate,
        };
    }

    return {
        startDate,
        endDate,
    };
}

function formatDateInputValue(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function formatStatsDate(date: Date): string {
    return date.toLocaleDateString("en-GB", {
        day: "numeric",
        month: "long",
        year: "numeric",
    });
}

function calculateAllTimeStats(sessions: Session[]): AllTimeStats {
    const endDate = new Date();

    if (sessions.length === 0) {
        return {
            startDate: null,
            endDate,
            totalSeconds: 0,
            sessionsCount: 0,
            activeDays: 0,
            calendarDays: 0,
            dailyAverageSeconds: 0,
            averageActiveDaySeconds: 0,
            activitiesCount: 0,
        };
    }

    const activeDateKeys = new Set<string>();
    const activityIDs = new Set<string>();

    let totalSeconds = 0;
    let earliestDate = parseDateKey(sessions[0].focus_day);

    for (const session of sessions) {
        const sessionDate = parseDateKey(session.focus_day);

        totalSeconds += session.duration_seconds;
        activeDateKeys.add(session.focus_day);

        if (session.activity) {
            activityIDs.add(session.activity.id);
        }

        if (sessionDate.getTime() < earliestDate.getTime()) {
            earliestDate = sessionDate;
        }
    }

    const activeDays = activeDateKeys.size;
    const calendarDays = getCalendarDaysCount(earliestDate, endDate);

    return {
        startDate: earliestDate,
        endDate,
        totalSeconds,
        sessionsCount: sessions.length,
        activeDays,
        calendarDays,
        dailyAverageSeconds:
            calendarDays > 0
                ? Math.round(totalSeconds / calendarDays)
                : 0,
        averageActiveDaySeconds:
            activeDays > 0
                ? Math.round(totalSeconds / activeDays)
                : 0,
        activitiesCount: activityIDs.size,
    };
}

function calculatePeriodStats(
    sessions: Session[],
    rangeStartDate: Date,
    rangeEndDate: Date,
): PeriodStats {
    const startDate = startOfDay(rangeStartDate);
    const endDate = endOfDay(rangeEndDate);

    const activeDateKeys = new Set<string>();

    const activitiesByID = new Map<
        string,
        {
            activityID: string | null;
            title: string;
            totalSeconds: number;
        }
    >();

    let totalSeconds = 0;
    let sessionsCount = 0;

    for (const session of sessions) {
        const sessionDate = parseDateKey(session.focus_day);

        if (!isDateInRange(sessionDate, startDate, endDate)) {
            continue;
        }

        totalSeconds += session.duration_seconds;
        sessionsCount += 1;
        activeDateKeys.add(session.focus_day);

        const activityKey =
            session.activity?.id ?? "no-activity";

        const currentActivity =
            activitiesByID.get(activityKey);

        if (currentActivity) {
            currentActivity.totalSeconds +=
                session.duration_seconds;
        } else {
            activitiesByID.set(activityKey, {
                activityID: session.activity?.id ?? null,
                title:
                    session.activity?.title ?? "No activity",
                totalSeconds: session.duration_seconds,
            });
        }
    }

    const activeDays = activeDateKeys.size;
    const calendarDays = getCalendarDaysCount(
        startDate,
        endDate,
    );

    const activities = Array.from(
        activitiesByID.values(),
    )
        .map((activity) => ({
            ...activity,
            percentage:
                totalSeconds > 0
                    ? Math.round(
                        (activity.totalSeconds /
                            totalSeconds) *
                        100,
                    )
                    : 0,
        }))
        .sort(
            (a, b) =>
                b.totalSeconds - a.totalSeconds,
        );

    return {
        startDate,
        endDate,
        totalSeconds,
        sessionsCount,
        activeDays,
        calendarDays,
        dailyAverageSeconds:
            calendarDays > 0
                ? Math.round(
                    totalSeconds / calendarDays,
                )
                : 0,
        averageActiveDaySeconds:
            activeDays > 0
                ? Math.round(
                    totalSeconds / activeDays,
                )
                : 0,
        activities,
    };
}


export function GlobalHistoryView() {
    const [sessions, setSessions] = useState<Session[]>([]);
    const [monthSort, setMonthSort] =
        useState<MonthSort>("newest");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [openedMonths, setOpenedMonths] =
        useState<Set<string>>(new Set());

    const [statsPeriod, setStatsPeriod] =
        useState<StatsPeriod>("week");

    const todayDateValue =
        formatDateInputValue(new Date());

    const defaultCustomStartDate = new Date();
    defaultCustomStartDate.setDate(
        defaultCustomStartDate.getDate() - 6,
    );

    const [customStartDate, setCustomStartDate] =
        useState(
            formatDateInputValue(
                defaultCustomStartDate,
            ),
        );

    const [customEndDate, setCustomEndDate] =
        useState(todayDateValue);

    const years = useMemo(() => {
        return buildGlobalHistory(sessions);
    }, [sessions]);

    const allTimeStats = useMemo(() => {
        return calculateAllTimeStats(sessions);
    }, [sessions]);

    const selectedDateRange = useMemo(() => {
        if (statsPeriod === "custom") {
            return {
                startDate:
                    parseDateKey(customStartDate),
                endDate:
                    parseDateKey(customEndDate),
            };
        }

        return getStatsDateRange(
            statsPeriod,
            sessions,
        );
    }, [
        statsPeriod,
        sessions,
        customStartDate,
        customEndDate,
    ]);

    const customRangeIsValid =
        statsPeriod !== "custom" ||
        selectedDateRange.startDate.getTime() <=
        selectedDateRange.endDate.getTime();

    const selectedPeriodStats = useMemo(() => {
        return calculatePeriodStats(
            sessions,
            selectedDateRange.startDate,
            selectedDateRange.endDate,
        );
    }, [
        sessions,
        selectedDateRange,
    ]);

    useEffect(() => {
        async function loadSessions() {
            setLoading(true);
            setError("");

            try {
                const result = await getAllSessions();
                setSessions(result);
            } catch (error) {
                setError(error instanceof Error ? error.message : "Failed to load history");
            } finally {
                setLoading(false);
            }
        }

        loadSessions();
    }, []);

    function toggleMonth(monthKey: string) {
        setOpenedMonths((current) => {
            const next = new Set(current);

            if (next.has(monthKey)) {
                next.delete(monthKey);
            } else {
                next.add(monthKey);
            }

            return next;
        });
    }

    function getSortedMonths(months: MonthSummary[]): MonthSummary[] {
        return [...months].sort((a, b) => {
            if (monthSort === "least-time") {
                return a.totalSeconds - b.totalSeconds;
            }

            if (monthSort === "most-time") {
                return b.totalSeconds - a.totalSeconds;
            }

            return b.monthIndex - a.monthIndex;
        });
    }

    return (
        <section className="card globalHistoryCard">
            <p className="eyebrow">Global History</p>

            <div className="allTimeStats">
                <div className="allTimeStatsHeader">
                    <div>
                        <p className="eyebrow">All-time stats</p>
                        <h1>Focus overview</h1>
                    </div>

                    {allTimeStats.startDate && (
                        <p className="allTimeStatsRange">
                            {formatStatsDate(allTimeStats.startDate)}
                            {" — "}
                            {formatStatsDate(allTimeStats.endDate)}
                        </p>
                    )}
                </div>

                <div className="allTimeStatsGrid">
                    <div>
                        <span>Total focus</span>
                        <strong>{formatDuration(allTimeStats.totalSeconds)}</strong>
                    </div>

                    <div>
                        <span>Active days</span>
                        <strong>{allTimeStats.activeDays}</strong>
                    </div>

                    <div>
                        <span>Daily average</span>
                        <strong>
                            {formatDuration(allTimeStats.dailyAverageSeconds)}
                        </strong>
                    </div>

                    <div>
                        <span>Average active day</span>
                        <strong>
                            {formatDuration(
                                allTimeStats.averageActiveDaySeconds,
                            )}
                        </strong>
                    </div>

                    <div>
                        <span>Sessions</span>
                        <strong>{allTimeStats.sessionsCount}</strong>
                    </div>

                    <div>
                        <span>Activities used</span>
                        <strong>{allTimeStats.activitiesCount}</strong>
                    </div>
                </div>
            </div>

            <section className="periodAnalytics">

                <div className="periodAnalyticsHeader">
                    <div>
                        <p className="eyebrow">Period analytics</p>
                        <h2>Selected period</h2>
                    </div>
                </div>

                <div className="periodSelector">
                    <button
                        type="button"
                        onClick={() => setStatsPeriod("week")}
                        className={
                            statsPeriod === "week"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        7 days
                    </button>

                    <button
                        type="button"
                        onClick={() => setStatsPeriod("month")}
                        className={
                            statsPeriod === "month"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        30 days
                    </button>

                    <button
                        type="button"
                        onClick={() => setStatsPeriod("quarter")}
                        className={
                            statsPeriod === "quarter"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        3 months
                    </button>

                    <button
                        type="button"
                        onClick={() =>
                            setStatsPeriod("half-year")
                        }
                        className={
                            statsPeriod === "half-year"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        6 months
                    </button>

                    <button
                        type="button"
                        onClick={() => setStatsPeriod("year")}
                        className={
                            statsPeriod === "year"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        1 year
                    </button>

                    <button
                        type="button"
                        onClick={() =>
                            setStatsPeriod("all-time")
                        }
                        className={
                            statsPeriod === "all-time"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        All time
                    </button>
                    <button
                        type="button"
                        onClick={() => setStatsPeriod("custom")}
                        className={
                            statsPeriod === "custom"
                                ? "periodButton periodButtonActive"
                                : "periodButton"
                        }
                    >
                        Custom
                    </button>
                </div>

                {statsPeriod === "custom" && (
                    <div className="customDateRange">
                        <label>
                            <span>From</span>

                            <input
                                type="date"
                                value={customStartDate}
                                max={customEndDate}
                                onChange={(event) =>
                                    setCustomStartDate(event.target.value)
                                }
                            />
                        </label>

                        <label>
                            <span>To</span>

                            <input
                                type="date"
                                value={customEndDate}
                                min={customStartDate}
                                max={todayDateValue}
                                onChange={(event) =>
                                    setCustomEndDate(event.target.value)
                                }
                            />
                        </label>
                    </div>
                )}

                {!customRangeIsValid && (
                    <p className="error">
                        Start date must not be later than end date.
                    </p>
                )}

                {customRangeIsValid && (
                    <>
                        <p className="periodDateRange">
                            {formatStatsDate(
                                selectedPeriodStats.startDate,
                            )}
                            {" — "}
                            {formatStatsDate(
                                selectedPeriodStats.endDate,
                            )}
                        </p>

                        <div className="periodStatsGrid">
                            <div>
                                <span>Total focus</span>
                                <strong>
                                    {formatDuration(
                                        selectedPeriodStats.totalSeconds,
                                    )}
                                </strong>
                            </div>

                            <div>
                                <span>Active days</span>
                                <strong>
                                    {selectedPeriodStats.activeDays}
                                </strong>
                            </div>

                            <div>
                                <span>Daily average</span>
                                <strong>
                                    {formatDuration(
                                        selectedPeriodStats.dailyAverageSeconds,
                                    )}
                                </strong>
                            </div>

                            <div>
                                <span>Average active day</span>
                                <strong>
                                    {formatDuration(
                                        selectedPeriodStats.averageActiveDaySeconds,
                                    )}
                                </strong>
                            </div>

                            <div>
                                <span>Sessions</span>
                                <strong>
                                    {selectedPeriodStats.sessionsCount}
                                </strong>
                            </div>
                        </div>

                        <div className="periodActivities">
                            <div className="periodActivitiesHeader">
                                <h3>Activities</h3>

                                <span>
                                    {selectedPeriodStats.activities.length}
                                </span>
                            </div>

                            {selectedPeriodStats.activities.length > 0 && (
                                <div className="periodActivitiesColumns">
                                    <span>#</span>
                                    <span>Activity</span>
                                    <span>Share</span>
                                    <span>Focused</span>
                                </div>
                            )}

                            {selectedPeriodStats.activities.length === 0 ? (
                                <p className="emptyState">
                                    No activity data for this period.
                                </p>
                            ) : (
                                <div className="periodActivitiesList">
                                    {selectedPeriodStats.activities.map(
                                        (activity, index) => (
                                            <div
                                                className="periodActivityItem"
                                                key={
                                                    activity.activityID ??
                                                    `no-activity-${index}`
                                                }
                                            >
                                                <span className="periodActivityIndex">
                                                    {String(index + 1).padStart(2, "0")}
                                                </span>

                                                <span
                                                    className="periodActivityTitle"
                                                    title={activity.title}
                                                >
                                                    {activity.title}
                                                </span>

                                                <span className="periodActivityPercentage">
                                                    {activity.percentage}%
                                                </span>

                                                <strong className="periodActivityDuration">
                                                    {formatDuration(
                                                        activity.totalSeconds,
                                                    )}
                                                </strong>
                                            </div>
                                        ),
                                    )}
                                </div>
                            )}
                        </div>
                    </>
                )}
            </section>

            {loading && <p className="emptyState">Loading global history...</p>}

            {error && <p className="error">{error}</p>}

            {!loading && !error && sessions.length === 0 && (
                <p className="emptyState">No saved focus sessions yet.</p>
            )}

            {!loading && !error && years.length > 0 && (
                <div className="globalYears">
                    {years.map((year) => (
                        <section className="globalYear" key={year.year}>
                            <div className="globalYearHeader">
                                <h2>{year.year}</h2>

                                <div className="historySort">
                                    <span>Sort months</span>

                                    <select
                                        value={monthSort}
                                        onChange={(event) =>
                                            setMonthSort(
                                                event.target.value as MonthSort,
                                            )
                                        }
                                    >
                                        <option value="newest">
                                            Newest first
                                        </option>

                                        <option value="most-time">
                                            Most focused time
                                        </option>

                                        <option value="least-time">
                                            Least focused time
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <div className="globalYearStats">
                                <p>Months worked: {year.monthsWorked}</p>
                                <p>Days worked: {year.daysWorked}</p>
                                <p>Time spent total: {formatDuration(year.totalSeconds)}</p>
                            </div>

                            <div className="globalMonths">
                                {getSortedMonths(year.months).map((month) => {
                                    const isOpen = openedMonths.has(month.monthKey);

                                    return (
                                        <section
                                            className={
                                                isOpen
                                                    ? "globalMonth globalMonthOpen"
                                                    : "globalMonth"
                                            }
                                            key={month.monthKey}
                                        >
                                            <button
                                                type="button"
                                                className="monthHeader"
                                                onClick={() => toggleMonth(month.monthKey)}
                                            >
                                                <span className="monthName">
                                                    {month.monthName}
                                                </span>

                                                <span className="monthSummary">
                                                    <span>
                                                        {month.daysWorked} days
                                                    </span>

                                                    <strong>
                                                        {formatDuration(month.totalSeconds)}
                                                    </strong>
                                                </span>

                                                <span
                                                    className={
                                                        isOpen
                                                            ? "monthChevron monthChevronOpen"
                                                            : "monthChevron"
                                                    }
                                                    aria-hidden="true"
                                                >
                                                    ›
                                                </span>
                                            </button>

                                            {isOpen && (
                                                <div className="monthWeeks">
                                                    {month.weeks.map((week) => (
                                                        <section className="monthWeek" key={week.weekKey}>
                                                            <h3>
                                                                Week {week.weekNumber}
                                                                <span>{formatDuration(week.totalSeconds)}</span>
                                                            </h3>

                                                            <div className="weekDays">
                                                                <div className="weekDaysHeader">
                                                                    <span>Day</span>
                                                                    <span>Sessions</span>
                                                                    <span>Worked</span>
                                                                </div>

                                                                {week.days.map((day) => (
                                                                    <div className="weekDayRow" key={day.dateKey}>
                                                                        <span>{formatDayLabel(day.date)}</span>
                                                                        <span>{day.sessionsCount}</span>
                                                                        <span>{formatDuration(day.totalSeconds)}</span>
                                                                    </div>
                                                                ))}
                                                            </div>
                                                        </section>
                                                    ))}
                                                </div>
                                            )}
                                        </section>
                                    );
                                })}
                            </div>
                        </section>
                    ))}
                </div>
            )}
        </section>
    );
}