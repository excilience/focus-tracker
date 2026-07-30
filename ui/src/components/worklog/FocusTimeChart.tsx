import { useMemo, useState } from "react";

import type { Session } from "../../api";

import type { ReactNode } from "react";

import {
    CartesianGrid,
    Line,
    LineChart,
    ResponsiveContainer,
    Tooltip,
    XAxis,
    YAxis,
} from "recharts";

function formatChartDuration(
    totalSeconds: number,
): string {
    if (totalSeconds === 0) {
        return "0m";
    }

    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor(
        (totalSeconds % 3600) / 60,
    );

    if (hours === 0) {
        return `${minutes}m`;
    }

    return `${hours}h ${minutes}m`;
}

function formatAxisHours(seconds: number) {
    return `${Math.round(seconds / 3600)}h`;
}

function formatChartDate(
    dateKey: string,
    period: FocusChartPeriod,
): string {
    const date = new Date(
        `${dateKey}T00:00:00`,
    );

    if (period === "month") {
        return date.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    if (period === "year") {
        return date.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    return date.toLocaleDateString("en-US", {
        month: "short",
        year: "numeric",
    });
}

type FocusChartTooltipPayload = {
    value?: number;
    payload?: {
        date: string;
        label: string;
        totalSeconds: number;
    };
};

type FocusChartTooltipProps = {
    active?: boolean;
    payload?: FocusChartTooltipPayload[];
    label?: ReactNode;
};

type FocusTimeChartProps = {
    sessions: Session[];
};

type FocusChartPeriod =
    | "month"
    | "year"
    | "max";

type FocusChartPoint = {
    date: string;
    totalSeconds: number;
};

function FocusChartTooltip({
    active,
    payload,
}: FocusChartTooltipProps) {
    if (!active || !payload || payload.length === 0) {
        return null;
    }

    const point = payload[0]?.payload;

    if (!point) {
        return null;
    }

    return (
        <div className="focusChartTooltip">
            <span className="focusChartTooltipLabel">
                {point.label}
            </span>

            <strong className="focusChartTooltipDuration">
                {formatChartDuration(
                    point.totalSeconds,
                )}
            </strong>
        </div>
    );
}

function buildSecondsByDate(
    sessions: Session[],
): Map<string, number> {
    const secondsByDate = new Map<string, number>();

    for (const session of sessions) {
        const currentSeconds =
            secondsByDate.get(session.focus_day) ?? 0;

        secondsByDate.set(
            session.focus_day,
            currentSeconds + session.duration_seconds,
        );
    }

    return secondsByDate;
}

function formatWeekRange(dateKey: string): string {
    const startDate = new Date(
        `${dateKey}T00:00:00`,
    );

    const endDate = new Date(startDate);
    endDate.setDate(endDate.getDate() + 6);

    const startLabel =
        startDate.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });

    const endLabel =
        endDate.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });

    return `${startLabel} – ${endLabel}`;
}


function getWeekStart(date: Date): Date {
    const weekStart = new Date(
        date.getFullYear(),
        date.getMonth(),
        date.getDate(),
    );

    const mondayOffset =
        (weekStart.getDay() + 6) % 7;

    weekStart.setDate(
        weekStart.getDate() - mondayOffset,
    );

    return weekStart;
}

function getWeekKey(dateKey: string): string {
    const date = new Date(
        `${dateKey}T00:00:00`,
    );

    return formatDateKey(getWeekStart(date));
}

function formatMonthKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(
        date.getMonth() + 1,
    ).padStart(2, "0");

    return `${year}-${month}`;
}

function buildYearlyChartData(
    sessions: Session[],
): FocusChartPoint[] {
    const secondsByWeek = new Map<string, number>();

    for (const session of sessions) {
        const weekKey = getWeekKey(
            session.focus_day,
        );

        const currentSeconds =
            secondsByWeek.get(weekKey) ?? 0;

        secondsByWeek.set(
            weekKey,
            currentSeconds +
            session.duration_seconds,
        );
    }

    const endWeek = getWeekStart(new Date());

    const currentWeek = new Date(endWeek);
    currentWeek.setDate(
        currentWeek.getDate() - 51 * 7,
    );

    const points: FocusChartPoint[] = [];

    for (let index = 0; index < 52; index += 1) {
        const weekKey =
            formatDateKey(currentWeek);

        points.push({
            date: weekKey,
            totalSeconds:
                secondsByWeek.get(weekKey) ?? 0,
        });

        currentWeek.setDate(
            currentWeek.getDate() + 7,
        );
    }

    return points;
}

function buildMaxChartData(
    sessions: Session[],
): FocusChartPoint[] {
    if (sessions.length === 0) {
        return [];
    }

    const secondsByMonth = new Map<string, number>();

    for (const session of sessions) {
        const monthKey =
            session.focus_day.slice(0, 7);

        const currentSeconds =
            secondsByMonth.get(monthKey) ?? 0;

        secondsByMonth.set(
            monthKey,
            currentSeconds + session.duration_seconds,
        );
    }

    const sortedDateKeys = Array.from(
        secondsByMonth.keys(),
    ).sort();

    const firstMonth = new Date(
        `${sortedDateKeys[0]}-01T00:00:00`,
    );

    const currentMonth = new Date(
        firstMonth.getFullYear(),
        firstMonth.getMonth(),
        1,
    );

    const endMonth = new Date();
    endMonth.setDate(1);

    const points: FocusChartPoint[] = [];

    while (currentMonth <= endMonth) {
        const monthKey =
            formatMonthKey(currentMonth);

        points.push({
            date: monthKey,
            totalSeconds:
                secondsByMonth.get(monthKey) ?? 0,
        });

        currentMonth.setMonth(
            currentMonth.getMonth() + 1,
        );
    }

    return points;
}

function formatDateKey(date: Date): string {
    const year = date.getFullYear();
    const month = String(
        date.getMonth() + 1,
    ).padStart(2, "0");
    const day = String(
        date.getDate(),
    ).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function formatTimestampLabel(
    timestamp: number,
    period: FocusChartPeriod,
): string {
    const date = new Date(timestamp);

    if (period === "month") {
        return date.toLocaleDateString("en-US", {
            month: "short",
            day: "numeric",
        });
    }

    if (period === "year") {
        return date.toLocaleDateString("en-US", {
            month: "short",
        });
    }

    return date.toLocaleDateString("en-US", {
        month: "short",
        year: "2-digit",
    });
}

function buildMonthlyChartData(
    sessions: Session[],
): FocusChartPoint[] {
    const secondsByDate = buildSecondsByDate(sessions);

    const points: FocusChartPoint[] = [];

    const endDate = new Date();
    const currentDate = new Date();

    currentDate.setDate(
        currentDate.getDate() - 29,
    );

    while (currentDate <= endDate) {
        const dateKey = formatDateKey(currentDate);

        points.push({
            date: dateKey,
            totalSeconds:
                secondsByDate.get(dateKey) ?? 0,
        });

        currentDate.setDate(
            currentDate.getDate() + 1,
        );
    }

    return points;
}

export function FocusTimeChart({
    sessions,
}: FocusTimeChartProps) {
    const [period, setPeriod] =
        useState<FocusChartPeriod>("month");

    const chartData = useMemo(() => {
        if (period === "year") {
            return buildYearlyChartData(sessions);
        }

        if (period === "max") {
            return buildMaxChartData(sessions);
        }

        return buildMonthlyChartData(sessions);
    }, [sessions, period]);

    const renderedChartData = useMemo(() => {
        return chartData.map((point) => {
            const date =
                period === "max"
                    ? new Date(
                        `${point.date}-01T00:00:00`,
                    )
                    : new Date(
                        `${point.date}T00:00:00`,
                    );

            return {
                ...point,
                timestamp: date.getTime(),
                label: formatChartDate(
                    point.date,
                    period,
                ),
            };
        });
    }, [chartData, period]);

    const yearlyTicks = useMemo(() => {
        if (period !== "year" || chartData.length === 0) {
            return undefined;
        }

        const firstDate = new Date(
            `${chartData[0].date}T00:00:00`,
        );

        const lastDate = new Date(
            `${chartData[chartData.length - 1].date}T00:00:00`,
        );

        const currentMonth = new Date(
            firstDate.getFullYear(),
            firstDate.getMonth() + 1,
            1,
        );

        const ticks: number[] = [];

        while (currentMonth <= lastDate) {
            ticks.push(currentMonth.getTime());

            currentMonth.setMonth(
                currentMonth.getMonth() + 1,
            );
        }

        return ticks;
    }, [chartData, period]);



    const chartStats = useMemo(() => {
        let totalSeconds = 0;
        let activePoints = 0;
        let bestPoint: FocusChartPoint | null = null;

        for (const point of chartData) {
            totalSeconds += point.totalSeconds;

            if (point.totalSeconds > 0) {
                activePoints += 1;
            }

            if (
                bestPoint === null ||
                point.totalSeconds > bestPoint.totalSeconds
            ) {
                bestPoint = point;
            }
        }

        const averageSeconds =
            activePoints > 0
                ? Math.round(totalSeconds / activePoints)
                : 0;

        return {
            totalSeconds,
            activePoints,
            averageSeconds,
            bestPoint,
        };
    }, [chartData]);

    const bestPointLabel =
        chartStats.bestPoint &&
            chartStats.bestPoint.totalSeconds > 0
            ? period === "year"
                ? formatWeekRange(
                    chartStats.bestPoint.date,
                )
                : formatChartDate(
                    chartStats.bestPoint.date,
                    period,
                )
            : "No data";

    const bestStatLabel =
        period === "month"
            ? "Peak Day"
            : period === "year"
                ? "Peak Week"
                : "Peak Month";

    const maxChartSeconds = Math.max(
        0,
        ...renderedChartData.map(
            (point) => point.totalSeconds,
        ),
    );

    const maxChartHours = Math.max(
        1,
        Math.ceil(maxChartSeconds / 3600),
    );

    const chartYAxisTicks = Array.from(
        { length: maxChartHours + 1 },
        (_, index) => index * 3600,
    );

    return (
        <section className="focusTimeChart">
            <div className="focusTimeChartHeader">
                <div>
                    <h2>Focus time</h2>
                </div>

                <div className="focusTimeChartPeriods">
                    <button
                        type="button"
                        onClick={() => setPeriod("month")}
                        className={
                            period === "month"
                                ? "focusChartPeriodButton focusChartPeriodButtonActive"
                                : "focusChartPeriodButton"
                        }
                    >
                        Month
                    </button>

                    <button
                        type="button"
                        onClick={() => setPeriod("year")}
                        className={
                            period === "year"
                                ? "focusChartPeriodButton focusChartPeriodButtonActive"
                                : "focusChartPeriodButton"
                        }
                    >
                        Year
                    </button>

                    <button
                        type="button"
                        onClick={() => setPeriod("max")}
                        className={
                            period === "max"
                                ? "focusChartPeriodButton focusChartPeriodButtonActive"
                                : "focusChartPeriodButton"
                        }
                    >
                        Max
                    </button>
                </div>
            </div>

            <div className="focusTimeChartStats">
                <div className="focusTimeChartStat">
                    <span>Total</span>

                    <strong>
                        {formatChartDuration(
                            chartStats.totalSeconds,
                        )}
                    </strong>
                </div>

                <div className="focusTimeChartStat">
                    <span>Active</span>

                    <strong>
                        {chartStats.activePoints}
                    </strong>
                </div>

                <div className="focusTimeChartStat">
                    <span>Average</span>

                    <strong>
                        {formatChartDuration(
                            chartStats.averageSeconds,
                        )}
                    </strong>
                </div>

                <div className="focusTimeChartStat">
                    <span>{bestStatLabel}</span>

                    <strong>
                        {bestPointLabel}
                    </strong>

                    {chartStats.bestPoint &&
                        chartStats.bestPoint.totalSeconds > 0 && (
                            <small>
                                {formatChartDuration(
                                    chartStats.bestPoint
                                        .totalSeconds,
                                )}
                            </small>
                        )}
                </div>
            </div>

            <div className="focusTimeChartBody">
                <ResponsiveContainer
                    width="100%"
                    height={320}
                >
                    <LineChart
                        data={renderedChartData}
                        margin={{
                            top: 16,
                            right: 16,
                            bottom: 8,
                            left: 0,
                        }}
                    >
                        <CartesianGrid
                            strokeDasharray="3 3"
                            vertical={false}
                        />

                        <XAxis
                            dataKey="timestamp"
                            type="number"
                            scale="time"
                            domain={["dataMin", "dataMax"]}
                            ticks={yearlyTicks}
                            tickFormatter={(value) => {
                                return formatTimestampLabel(
                                    Number(value),
                                    period,
                                );
                            }}
                            tickLine={false}
                            axisLine={false}
                            minTickGap={20}
                        />

                        <YAxis
                            ticks={chartYAxisTicks}
                            domain={[0, maxChartHours * 3600]}
                            tickFormatter={formatAxisHours}
                            tickLine={false}
                            axisLine={false}
                            width={30}
                        />

                        <Tooltip
                            content={<FocusChartTooltip />}
                            cursor={{
                                strokeDasharray: "4 4",
                            }}
                        />

                        <Line
                            type="monotone"
                            dataKey="totalSeconds"
                            name="Focus time"
                            stroke="currentColor"
                            strokeWidth={2}
                            dot={false}
                            activeDot={{ r: 5 }}
                        />
                    </LineChart>
                </ResponsiveContainer>
            </div>
        </section>
    );
}