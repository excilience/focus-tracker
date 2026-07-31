import { useMemo, useState } from "react";

import type { Session } from "../../api";

type FocusHeatmapProps = {
    sessions: Session[];
    dayStartHour: number;
};

type HeatmapDay = {
    date: string;
    totalSeconds: number;
    level: number;
    isFuture: boolean;
};

type HeatmapCell =
    | {
        type: "empty";
        key: string;
    }
    | {
        type: "day";
        key: string;
        day: HeatmapDay;
    };

type HeatmapMonthLabel = {
    month: string;
    column: number;
};


function formatDateKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function formatDisplayDate(dateKey: string): string {
    const [year, month, day] = dateKey.split("-");
    return `${day}-${month}-${year}`;
}

function formatDuration(totalSeconds: number): string {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);

    if (hours > 0 && minutes > 0) return `${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h`;
    if (minutes > 0) return `${minutes}m`;
    return "0m";
}

function buildHeatmapDays(
    sessions: Session[],
    year: number,
    dayStartHour: number,
): HeatmapDay[] {
    const secondsByDate = new Map<string, number>();
    const todayDateKey = getCurrentFocusDateKey(dayStartHour);


    for (const session of sessions) {
        const dateKey = session.focus_day;
        if (!dateKey) {
            continue;
        }

        const currentSeconds =
            secondsByDate.get(dateKey) ?? 0;

        secondsByDate.set(
            dateKey,
            currentSeconds +
            session.duration_seconds,
        );
    }

    const days: HeatmapDay[] = [];
    const currentDate = new Date(year, 0, 1);
    const endDate = new Date(year, 11, 31);

    while (currentDate <= endDate) {
        const dateKey = formatDateKey(currentDate);

        const totalSeconds =
            secondsByDate.get(dateKey) ?? 0;


        days.push({
            date: dateKey,
            totalSeconds,
            level: getHeatmapLevel(totalSeconds),
            isFuture: dateKey > todayDateKey,
        });

        currentDate.setDate(
            currentDate.getDate() + 1,
        );
    }

    return days;
}

function buildHeatmapMonthLabels(
    days: HeatmapDay[],
): HeatmapMonthLabel[] {
    if (days.length === 0) {
        return [];
    }

    const labels: HeatmapMonthLabel[] = [];
    const firstDate = new Date(`${days[0].date}T00:00:00`);
    const firstDayOffset =
        (firstDate.getDay() + 6) % 7;

    let previousMonth = -1;

    for (let index = 0; index < days.length; index += 1) {
        const currentDate = new Date(
            `${days[index].date}T00:00:00`,
        );

        const month = currentDate.getMonth();

        if (month === previousMonth) {
            continue;
        }

        const absoluteIndex =
            firstDayOffset + index;

        const column =
            Math.floor(absoluteIndex / 7) + 1;

        labels.push({
            month: currentDate.toLocaleString("en-US", {
                month: "short",
            }),
            column,
        });

        previousMonth = month;
    }

    return labels;
}

function getHeatmapLevel(totalSeconds: number): number {
    if (totalSeconds === 0) {
        return 0;
    }

    if (totalSeconds < 30 * 60) {
        return 1;
    }

    if (totalSeconds < 60 * 60) {
        return 2;
    }

    if (totalSeconds < 2 * 60 * 60) {
        return 3;
    }

    return 4;
}

function buildHeatmapCells(
    days: HeatmapDay[],
): HeatmapCell[] {
    if (days.length === 0) {
        return [];
    }

    const cells: HeatmapCell[] = [];

    const firstDate = new Date(
        `${days[0].date}T00:00:00`,
    );

    const firstDayIndex =
        (firstDate.getDay() + 6) % 7;

    for (let index = 0; index < firstDayIndex; index += 1) {
        cells.push({
            type: "empty",
            key: `empty-start-${index}`,
        });
    }

    for (const day of days) {
        cells.push({
            type: "day",
            key: day.date,
            day,
        });
    }

    return cells;
}

function getAvailableYears(
    sessions: Session[],
): number[] {
    const years = new Set<number>();

    for (const session of sessions) {
        const dateKey = session.focus_day;

        if (!dateKey) {
            continue;
        }

        const year = Number(
            dateKey.slice(0, 4),
        );

        if (!Number.isNaN(year)) {
            years.add(year);
        }
    }

    return Array.from(years).sort(
        (a, b) => b - a,
    );
}

function getCurrentFocusDateKey(dayStartHour: number): string {
    const date = new Date();
    date.setHours(date.getHours() - dayStartHour);

    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

export function FocusHeatmap({
    sessions, dayStartHour,
}: FocusHeatmapProps) {
    const availableYears = useMemo(() => {
        return getAvailableYears(sessions);
    }, [sessions]);

    const [selectedYear, setSelectedYear] =
        useState(() => new Date().getFullYear());

    const heatmapDays = useMemo(() => {
        return buildHeatmapDays(
            sessions,
            selectedYear,
            dayStartHour,
        );
    }, [sessions, selectedYear, dayStartHour]);

    const monthLabels = useMemo(() => {
        return buildHeatmapMonthLabels(heatmapDays);
    }, [heatmapDays]);

    const heatmapCells = useMemo(() => {
        return buildHeatmapCells(heatmapDays);
    }, [heatmapDays]);

    const selectedYearStats = useMemo(() => {
        let activeDays = 0;
        let totalSeconds = 0;

        for (const day of heatmapDays) {
            totalSeconds += day.totalSeconds;

            if (day.totalSeconds > 0) {
                activeDays += 1;
            }
        }

        return {
            activeDays,
            totalSeconds,
        };
    }, [heatmapDays]);

    const todayDateKey = formatDateKey(new Date());

    return (
        <section className="focusHeatmap">
            <div className="focusHeatmapHeader">
                <div>
                    <h2>Focus heatmap</h2>
                </div>

                <p className="focusHeatmapSummary">
                    {selectedYearStats.activeDays} active days
                    {" · "}
                    {formatDuration(selectedYearStats.totalSeconds)} focused
                </p>

                <select
                    value={selectedYear}
                    onChange={(event) => {
                        setSelectedYear(Number(event.target.value));
                    }}
                    aria-label="Select heatmap year"
                >
                    {availableYears.map((year) => (
                        <option key={year} value={year}>
                            {year}
                        </option>
                    ))}
                </select>
            </div>


            <div className="focusHeatmapBody">
                <div className="focusHeatmapWeekdays">
                    <span>Mon</span>
                    <span></span>
                    <span>Wed</span>
                    <span></span>
                    <span>Fri</span>
                    <span></span>
                    <span>Sun</span>
                </div>

                <div className="focusHeatmapLegend">
                    <span>Less</span>

                    <div className="heatmapCell heatmapLevel0" />
                    <div className="heatmapCell heatmapLevel1" />
                    <div className="heatmapCell heatmapLevel2" />
                    <div className="heatmapCell heatmapLevel3" />
                    <div className="heatmapCell heatmapLevel4" />

                    <span>More</span>
                </div>

                <div className="focusHeatmapContent">
                    <div className="focusHeatmapMonths">
                        {monthLabels.map((label) => (
                            <span
                                key={`${selectedYear}-${label.month}`}
                                style={{
                                    gridColumnStart: label.column,
                                }}
                            >
                                {label.month}
                            </span>
                        ))}
                    </div>

                    <div className="focusHeatmapGrid">
                        {heatmapCells.map((cell) => {
                            if (cell.type === "empty") {
                                return (
                                    <div
                                        key={cell.key}
                                        className="heatmapCell heatmapCellEmpty"
                                    />
                                );
                            }

                            return (
                                <div
                                    key={cell.key}
                                    className={[
                                        "heatmapCell",
                                        `heatmapLevel${cell.day.level}`,
                                        cell.day.date === todayDateKey
                                            ? "heatmapCellToday"
                                            : "",
                                        cell.day.isFuture
                                            ? "heatmapCellFuture"
                                            : "",
                                    ]
                                        .filter(Boolean)
                                        .join(" ")}
                                    title={
                                        cell.day.isFuture
                                            ? undefined
                                            : `${formatDisplayDate(cell.day.date)}: ${formatDuration(cell.day.totalSeconds)}`
                                    }
                                />
                            );
                        })}
                    </div>
                </div>
            </div>
        </section>
    );
}