import { useEffect, useState } from "react";
import {
  createActivity,
  getActiveSession,
  getActivities,
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
  type Activity,
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

function calculateActivitySelectWidth(text: string): number {
  const canvas = document.createElement("canvas");
  const context = canvas.getContext("2d");

  if (!context) {
    return 140;
  }

  context.font = "700 15px Inter, sans-serif";

  const textWidth = context.measureText(text).width;

  const horizontalPadding = 52;
  const minWidth = 160;
  const maxWidth = 260;

  return Math.min(
    maxWidth,
    Math.max(minWidth, Math.ceil(textWidth + horizontalPadding))
  );
}

function App() {
  const [apiStatus, setApiStatus] = useState<"checking" | "ok" | "error">("checking");
  const [activeSession, setActiveSession] = useState<ActiveSession | null>(null);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [selectedActivityID, setSelectedActivityID] = useState("");
  const [isActivityModalOpen, setIsActivityModalOpen] = useState(false);
  const [activityTitleInput, setActivityTitleInput] = useState("");
  const [activitySaving, setActivitySaving] = useState(false);
  const [activitySelectWidth, setActivitySelectWidth] = useState(140);
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

  function openActivityModal() {
    setActivityTitleInput("");
    setError("");
    setIsActivityModalOpen(true);
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

    const activitiesResult = await getActivities();
    setActivities(activitiesResult);

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

  async function startFocusSession() {
    const activityID = selectedActivityID || undefined;

    await runAction(() => startSession(activityID));
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

  async function saveActivity() {
    const title = activityTitleInput.trim();

    if (!title) {
      setError("Activity title is required");
      return;
    }

    setActivitySaving(true);
    setError("");

    try {
      const createdActivity = await createActivity(title);

      setActivities((currentActivities) => [
        ...currentActivities,
        createdActivity,
      ]);

      setSelectedActivityID(createdActivity.id);
      setActivityTitleInput("");
      setIsActivityModalOpen(false);
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "Failed to create activity",
      );
    } finally {
      setActivitySaving(false);
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
      const secondsSinceSync = Math.floor(
        (Date.now() - activeSessionSyncedAt) / 1000
      );

      setLiveFocusedSeconds(
        activeSession.focused_seconds + Math.max(0, secondsSinceSync)
      );
    };

    updateLiveSeconds();

    const intervalID = window.setInterval(updateLiveSeconds, 1000);

    return () => {
      window.clearInterval(intervalID);
    };
  }, [activeSession, activeSessionSyncedAt]);

  const selectedActivity = activities.find(
    (activity) => activity.id === selectedActivityID
  );

  useEffect(() => {
    const selectedText = selectedActivity?.title ?? "No activity";

    setActivitySelectWidth(
      calculateActivitySelectWidth(selectedText)
    );
  }, [selectedActivity?.title]);


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

          {!hasActiveSession && (
            <div
              className="activityPicker"
              style={{ width: `${activitySelectWidth + 42}px` }}
            >
              <label className="activitySelector activitySelectorAboveRing">
                <span className="activitySelectorLabel">Activity</span>

                <select
                  className="activitySelectorInput"
                  value={selectedActivityID}
                  onChange={(event) => setSelectedActivityID(event.target.value)}
                  disabled={loading || activitySaving}
                >
                  <option value="">No activity</option>

                  {activities.map((activity) => (
                    <option key={activity.id} value={activity.id}>
                      {activity.title}
                    </option>
                  ))}
                </select>
              </label>

              <button
                type="button"
                className="activityCreateButton"
                onClick={openActivityModal}
                disabled={loading || activitySaving}
                aria-label="Create activity"
                title="Create activity"
              >
                +
              </button>
            </div>
          )}

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
                  <span
                    className="activeActivityLabel"
                    title={activeSession.activity?.title ?? "No activity"}
                  >
                    {activeSession.activity?.title ?? "No activity"}
                  </span>

                  <strong className="focusMainValue">
                    {formatTimer(liveFocusedSeconds)}
                  </strong>

                  <span
                    className={
                      isPaused
                        ? "focusSubLabel focusStatus focusStatusPaused"
                        : "focusSubLabel focusStatus"
                    }
                  >
                    {isPaused ? "Paused" : "Focusing"}
                  </span>
                </>
              ) : (
                <>
                  <span className="focusSubLabel">Goal</span>

                  <strong className="focusGoalValue">
                    {formatGoal(goal)}
                  </strong>
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
                onClick={() => void startFocusSession()}
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

      {isActivityModalOpen && (
        <div
          className="modalOverlay"
          role="presentation"
          onMouseDown={() => {
            if (!activitySaving) {
              setIsActivityModalOpen(false);
            }
          }}
        >
          <form
            className="goalModal"
            onMouseDown={(event) => event.stopPropagation()}
            onSubmit={(event) => {
              event.preventDefault();
              void saveActivity();
            }}
          >
            <h2 className="goalModalTitle">New Activity</h2>

            <label className="goalModalField">
              <span className="goalModalLabel">Title</span>

              <input
                className="goalModalInput"
                value={activityTitleInput}
                onChange={(event) => setActivityTitleInput(event.target.value)}
                placeholder="For example: Reading"
                disabled={activitySaving}
                aria-label="Activity title"
                autoFocus
              />
            </label>

            <button
              type="submit"
              className="goalModalSaveButton"
              disabled={activitySaving || !activityTitleInput.trim()}
            >
              {activitySaving ? "Creating..." : "Create"}
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