import { useMemo, useState } from "react";

import type { Session } from "../../api";

type FocusHeatmapProps = {
    sessions: Session[];
};

function getAvailableYears(sessions: Session[]): number[] {
    const years = new Set<number>();

    for (const session of sessions) {
        const year = Number(
            session.focus_day.slice(0, 4),
        );

        if (!Number.isNaN(year)) {
            years.add(year);
        }
    }

    years.add(new Date().getFullYear());

    return Array.from(years).sort(
        (first, second) => second - first,
    );
}

export function FocusHeatmap({
    sessions,
}: FocusHeatmapProps) {
    const availableYears = useMemo(() => {
        return getAvailableYears(sessions);
    }, [sessions]);

    const [selectedYear, setSelectedYear] =
        useState(() => new Date().getFullYear());

    return (
        <section className="focusHeatmap">
            <div className="focusHeatmapHeader">
                <div>
                    <p className="eyebrow">
                        Activity
                    </p>

                    <h2>Focus heatmap</h2>
                </div>

                <div className="focusHeatmapYears">
                    {availableYears.map((year) => (
                        <button
                            type="button"
                            key={year}
                            onClick={() =>
                                setSelectedYear(year)
                            }
                            className={
                                selectedYear === year
                                    ? "heatmapYearButton heatmapYearButtonActive"
                                    : "heatmapYearButton"
                            }
                        >
                            {year}
                        </button>
                    ))}
                </div>
            </div>

            <p className="emptyState">
                Selected year: {selectedYear}
            </p>
        </section>
    );
}