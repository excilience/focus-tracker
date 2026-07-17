import { useEffect, useState } from "react";
import {
  getActiveSession,
  getHealth,
  getSettings,
  pauseSession,
  resumeSession,
  startSession,
  stopSession,
  getGoal,
  updateGoal,
  updateSettings,
  type GoalResponse,
  type ActiveSession,
} from "./api";
import "./App.css";
import { QuickHistoryView } from "./QuickHistoryView";
import { SessionsView } from "./SessionsView"
import { GlobalHistoryView } from "./GlobalHistoryView";


function formatTimer(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  return [
    String(hours).padStart(2, "0"),
    String(minutes).padStart(2, "0"),
    String(seconds).padStart(2, "0"),
  ].join(":");
}

function formatFocusedTime(totalSeconds: number): string {
  const totalMinutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  if (hours === 0) {
    return `${minutes}m`;
  }

  if (minutes === 0) {
    return `${hours}h`;
  }

  return `${hours}h ${minutes}m`;
}

function formatGoal(goal: GoalResponse | null): string {
  if (!goal) {
    return "—";
  }

  const totalMinutes = Math.round(goal.goal_seconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  if (hours === 0) {
    return `${minutes}m`;
  }

  if (minutes === 0) {
    return `${hours}h`;
  }

  return `${hours}h ${minutes}m`;
}

function App() {
  const [apiStatus, setApiStatus] = useState<"checking" | "ok" | "error">("checking");
  const [activeSession, setActiveSession] = useState<ActiveSession | null>(null);
  const [activeSessionSyncedAt, setActiveSessionSyncedAt] = useState<number | null>(null);
  const [goal, setGoal] = useState<GoalResponse | null>(null);
  const [goalInput, setGoalInput] = useState("");
  const [goalSaving, setGoalSaving] = useState(false);
  const [isGoalModalOpen, setIsGoalModalOpen] = useState(false);
  const [dayStartHourInput, setDayStartHourInput] = useState("");
  const [activeTab, setActiveTab] = useState<
    "dashboard" | "history" | "sessions" | "globalHistory"
  >("dashboard");
  const [liveFocusedSeconds, setLiveFocusedSeconds] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function openGoalModal() {
    if (goal) {
      setGoalInput(goal.goal_human.replaceAll(" ", ""));
    }

    setError("");
    setIsGoalModalOpen(true);
  }

  async function refresh() {
    setError("");

    try {
      await getHealth();
      setApiStatus("ok");
    } catch {
      setApiStatus("error");
      setActiveSession(null);
      setActiveSessionSyncedAt(null);
      setLiveFocusedSeconds(0);
      return;
    }

    const session = await getActiveSession();
    setActiveSession(session);
    setActiveSessionSyncedAt(session ? Date.now() : null);
    setLiveFocusedSeconds(session?.focused_seconds ?? 0);

    const goalResult = await getGoal();
    setGoal(goalResult);
    setGoalInput(goalResult.goal_human.replaceAll(" ", ""));

    const settingsResult = await getSettings();
    setDayStartHourInput(String(settingsResult.day_start_hour));
  }

  async function runAction(action: () => Promise<unknown>) {
    setLoading(true);
    setError("");

    try {
      await action();
      await refresh();
    } catch (error) {
      setError(error instanceof Error ? error.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  async function saveSettings() {
    const parsedDayStartHour = Number(dayStartHourInput);

    if (!Number.isInteger(parsedDayStartHour) || parsedDayStartHour < 0 || parsedDayStartHour > 23) {
      setError("Day start hour must be between 0 and 23");
      return;
    }

    setGoalSaving(true);
    setError("");

    try {
      const updatedGoal = await updateGoal(goalInput);
      const updatedSettings = await updateSettings(parsedDayStartHour);


      setGoal(updatedGoal);


      setGoalInput(updatedGoal.goal_human.replaceAll(" ", ""));
      setDayStartHourInput(String(updatedSettings.day_start_hour));

      setIsGoalModalOpen(false);

      await refresh();
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to update goal");
    } finally {
      setGoalSaving(false);
    }
  }


  useEffect(() => {
    void refresh();
  }, []);


  useEffect(() => {
    if (!isGoalModalOpen) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsGoalModalOpen(false);
      }
    };

    window.addEventListener("keydown", handleKeyDown);

    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [isGoalModalOpen]);

  useEffect(() => {
    if (!activeSession) {
      setLiveFocusedSeconds(0);
      return;
    }

    if (activeSession.is_paused) {
      setLiveFocusedSeconds(activeSession.focused_seconds);
      return;
    }

    if (activeSessionSyncedAt === null) {
      setLiveFocusedSeconds(activeSession.focused_seconds);
      return;
    }

    const updateLiveSeconds = () => {
      const secondsSinceSync = Math.floor((Date.now() - activeSessionSyncedAt) / 1000);
      setLiveFocusedSeconds(activeSession.focused_seconds + Math.max(0, secondsSinceSync));
    };

    updateLiveSeconds();

    const intervalID = window.setInterval(updateLiveSeconds, 1000);

    return () => {
      window.clearInterval(intervalID);
    };
  }, [activeSession, activeSessionSyncedAt]);

  const hasActiveSession = activeSession !== null;
  const isPaused = activeSession?.is_paused ?? false;
  const liveDailyFocusedSeconds =
    (goal?.focused_seconds ?? 0) +
    (hasActiveSession ? liveFocusedSeconds : 0);

  const progressRatio =
    goal && goal.goal_seconds > 0
      ? Math.min(liveDailyFocusedSeconds / goal.goal_seconds, 1)
      : 0;

  const circleSize = 320;
  const circleStroke = 21;
  const circleCenter = circleSize / 2;
  const circleRadius = circleCenter - circleStroke / 23;
  const circleLength = 2 * Math.PI * circleRadius;
  const circleOffset = circleLength * (1 - progressRatio);

  return (
    <main className="page">
      <div className="tabs">
        <button
          className={activeTab === "dashboard" ? "tab activeTab" : "tab"}
          onClick={() => setActiveTab("dashboard")}
        >
          Dashboard
        </button>

        <button
          className={activeTab === "history" ? "tab activeTab" : "tab"}
          onClick={() => setActiveTab("history")}
        >
          History
        </button>

        <button
          className={activeTab === "sessions" ? "tab activeTab" : "tab"}
          onClick={() => setActiveTab("sessions")}
        >
          Sessions
        </button>

        <button
          className={activeTab === "globalHistory" ? "tab activeTab" : "tab"}
          onClick={() => setActiveTab("globalHistory")}
        >
          Worklog
        </button>
      </div>

      {activeTab === "dashboard" && (
        <section className="focusCard">
          <button
            type="button"
            className="goalEditButton"
            onClick={openGoalModal}
            disabled={goalSaving}
            aria-label="Edit daily goal"
          >
            <span aria-hidden="true">✎</span>
          </button>

          <div className="focusRingWrap">
            <svg className="focusRing" viewBox={`0 0 ${circleSize} ${circleSize}`}>
              <circle
                className="focusRingTrack"
                cx={circleCenter}
                cy={circleCenter}
                r={circleRadius}
                strokeWidth={circleStroke}
                fill="none"
              />

              <circle
                className={
                  hasActiveSession && !isPaused
                    ? "focusRingProgress focusRingActive"
                    : "focusRingProgress"
                }
                cx={circleCenter}
                cy={circleCenter}
                r={circleRadius}
                strokeWidth={circleStroke}
                strokeDasharray={circleLength}
                strokeDashoffset={circleOffset}
                fill="none"
              />
            </svg>

            <div className="focusRingContent">
              {hasActiveSession ? (
                <>
                  <strong className="focusMainValue">
                    {formatTimer(liveFocusedSeconds)}
                  </strong>
                  <span className="focusSubLabel">
                    {isPaused ? "Paused" : "Focusing"}
                  </span>
                </>
              ) : (
                <>
                  <span className="focusSubLabel">Goal</span>
                  <strong className="focusGoalValue">{formatGoal(goal)}</strong>
                </>
              )}
            </div>
          </div>

          <div className="completedBlock">
            <span className="completedLabel">Completed today</span>

            <strong className="completedValue">
              {formatFocusedTime(liveDailyFocusedSeconds)}
            </strong>
          </div>

          {apiStatus === "error" && (
            <p className="error focusError">API is unavailable</p>
          )}

          {error && <p className="error focusError">{error}</p>}

          <div className="focusControls">
            <div className="controlItem">
              <button
                className="roundButton refreshButton"
                onClick={refresh}
                disabled={loading}
                aria-label="Refresh"
              >
                <span className="iconRefresh">↺</span>
              </button>

              <span className="controlLabel">Refresh</span>
            </div>

            {!hasActiveSession ? (
              <button
                className="mainRoundButton"
                onClick={() => runAction(startSession)}
                disabled={loading}
                aria-label="Start"
              >
                <span className="iconPlay" />
              </button>
            ) : isPaused ? (
              <button
                className="mainRoundButton"
                onClick={() => runAction(resumeSession)}
                disabled={loading}
                aria-label="Resume"
              >
                <span className="iconPlay" />
              </button>
            ) : (
              <button
                className="mainRoundButton"
                onClick={() => runAction(pauseSession)}
                disabled={loading}
                aria-label="Pause"
              >
                <span className="iconPause" />
              </button>
            )}

            <div className="controlItem">
              <button
                className="roundButton stopButton"
                onClick={() => runAction(stopSession)}
                disabled={loading || !hasActiveSession}
                aria-label="Stop"
              >
                <span className="iconStop" />
              </button>

              <span className="controlLabel">Stop</span>
            </div>
          </div>
        </section>
      )}
      {isGoalModalOpen && (
        <div
          className="modalOverlay"
          role="presentation"
          onMouseDown={() => setIsGoalModalOpen(false)}
        >
          <form
            className="goalModal"
            onMouseDown={(event) => event.stopPropagation()}
            onSubmit={(event) => {
              event.preventDefault();
              void saveSettings();
            }}
          >
            <h2 className="goalModalTitle">Settings</h2>

            <label className="goalModalField">
              <span className="goalModalLabel">Daily goal</span>
              <input
                className="goalModalInput goalModalInputGoal"
                value={goalInput}
                onChange={(event) => setGoalInput(event.target.value)}
                placeholder="For example: 2h or 1h30m"
                disabled={goalSaving}
                aria-label="Daily focus goal"
                autoFocus
              />
            </label>

            <label className="goalModalField">
              <span className="goalModalLabel">Day start hour</span>

              <div className="hourInputRow">
                <input
                  className="goalModalInput goalModalInputHour"
                  type="number"
                  min={0}
                  max={23}
                  value={dayStartHourInput}
                  onChange={(event) => setDayStartHourInput(event.target.value)}
                  placeholder="0"
                  disabled={goalSaving}
                  aria-label="Day start hour"
                />

                <span className="hourSuffix">:00</span>
              </div>

              <span className="goalModalHint">
                Sessions before this time count toward the previous day.
              </span>
            </label>

            <button
              type="submit"
              className="goalModalSaveButton"
              disabled={goalSaving || !goalInput.trim() || !dayStartHourInput.trim()}
            >
              {goalSaving ? "Saving..." : "Save"}
            </button>
          </form>
        </div>
      )}

      {activeTab === "history" && <QuickHistoryView />}
      {activeTab === "sessions" && <SessionsView />}
      {activeTab === "globalHistory" && <GlobalHistoryView />}
    </main>
  );
}

export default App;