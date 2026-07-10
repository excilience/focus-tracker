import { useEffect, useState } from "react";
import {
  getActiveSession,
  getHealth,
  pauseSession,
  resumeSession,
  startSession,
  stopSession,
  getGoal,
  updateGoal,
  type GoalResponse,
  type ActiveSession,
} from "./api";
import "./App.css";
import { QuickHistoryView } from "./QuickHistoryView";
import { SessionsView } from "./SessionsView"

function formatSeconds(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m ${seconds}s`;
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }

  return `${seconds}s`;
}



function App() {
  const [apiStatus, setApiStatus] = useState<"checking" | "ok" | "error">("checking");
  const [activeSession, setActiveSession] = useState<ActiveSession | null>(null);
  const [activeSessionSyncedAt, setActiveSessionSyncedAt] = useState<number | null>(null);
  const [goal, setGoal] = useState<GoalResponse | null>(null);
  const [goalInput, setGoalInput] = useState("");
  const [goalSaving, setGoalSaving] = useState(false);
  const [activeTab, setActiveTab] = useState<"dashboard" | "history" | "sessions">("dashboard");
  const [liveFocusedSeconds, setLiveFocusedSeconds] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

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

  async function saveGoal() {
    setGoalSaving(true);
    setError("");

    try {
      const updatedGoal = await updateGoal(goalInput);
      setGoal(updatedGoal);
      setGoalInput(updatedGoal.goal_human.replaceAll(" ", ""));
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to update goal");
    } finally {
      setGoalSaving(false);
    }
  }

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
      </div>

      {activeTab === "dashboard" && (
        <section className="card">
          <p className="eyebrow">Focus Tracker</p>
          <h1>Dashboard</h1>

          <div className="statusGrid">
            <div>
              <span className="label">API</span>
              <strong className={apiStatus === "ok" ? "good" : "bad"}>
                {apiStatus}
              </strong>
            </div>

            <div>
              <span className="label">Active session</span>
              <strong>{hasActiveSession ? "yes" : "no"}</strong>
            </div>

            <div>
              <span className="label">State</span>
              <strong>
                {!hasActiveSession ? "idle" : isPaused ? "paused" : "running"}
              </strong>
            </div>
          </div>

          {activeSession ? (
            <div className="sessionBox">
              <div>
                <span className="label">Started</span>
                <strong>{activeSession.start}</strong>
              </div>

              <div>
                <span className="label">Focused</span>
                <strong>{formatSeconds(liveFocusedSeconds)}</strong>
              </div>

              <div>
                <span className="label">Focused seconds</span>
                <strong>{liveFocusedSeconds}</strong>
              </div>
            </div>
          ) : (
            <p className="emptyState">No active focus session yet.</p>
          )}

          {goal && (
            <div className="goalBox">
              <div className="goalHeader">
                <div>
                  <span className="label">Daily Goal</span>
                  <strong>{goal.goal_human}</strong>
                </div>

                <div>
                  <span className="label">Progress</span>
                  <strong>{goal.percent}%</strong>
                </div>
              </div>

              <div className="goalGrid">
                <div>
                  <span className="label">Focused today</span>
                  <strong>{goal.focused_human}</strong>
                </div>

                <div>
                  <span className="label">Remaining</span>
                  <strong>{goal.remaining_human}</strong>
                </div>
              </div>

              <div className="goalEditor">
                <input
                  className="goalInput"
                  value={goalInput}
                  onChange={(event) => setGoalInput(event.target.value)}
                  placeholder="2h or 1h30m"
                  disabled={goalSaving}
                />

                <button onClick={saveGoal} disabled={goalSaving}>
                  Save Goal
                </button>
              </div>
            </div>
          )}

          {error && <p className="error">{error}</p>}

          <div className="buttons">
            <button
              onClick={() => runAction(startSession)}
              disabled={loading || hasActiveSession}
            >
              Start
            </button>

            <button
              onClick={() => runAction(pauseSession)}
              disabled={loading || !hasActiveSession || isPaused}
            >
              Pause
            </button>

            <button
              onClick={() => runAction(resumeSession)}
              disabled={loading || !hasActiveSession || !isPaused}
            >
              Resume
            </button>

            <button
              className="danger"
              onClick={() => runAction(stopSession)}
              disabled={loading || !hasActiveSession}
            >
              Stop
            </button>
          </div>

          <button className="secondary" onClick={refresh} disabled={loading}>
            Refresh
          </button>
        </section>
      )}
      {activeTab === "history" && <QuickHistoryView />}
      {activeTab === "sessions" && <SessionsView />}
    </main>
  );
}

export default App;