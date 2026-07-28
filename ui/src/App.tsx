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
  updateActivity,
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
  const [managedActivities, setManagedActivities] = useState<Activity[]>([]);
  const [isActivityManagerOpen, setIsActivityManagerOpen] = useState(false);
  const [activityManagerLoading, setActivityManagerLoading] = useState(false);
  const [activityManagerError, setActivityManagerError] = useState("");
  const [isArchiveOpen, setIsArchiveOpen] = useState(false);
  const [isActivityModalOpen, setIsActivityModalOpen] = useState(false);
  const [activityTitleInput, setActivityTitleInput] = useState("");
  const [activitySaving, setActivitySaving] = useState(false);
  const [activitySelectWidth, setActivitySelectWidth] = useState(140);
  const [activityError, setActivityError] = useState("");
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
  const activeManagedActivities = managedActivities.filter(
    (activity) => !activity.is_archived,
  );
  const archivedManagedActivities = managedActivities.filter(
    (activity) => activity.is_archived,
  );
  const [activityManagerSavingID, setActivityManagerSavingID] =
    useState<string | null>(null);
  const [editingActivityID, setEditingActivityID] =
    useState<string | null>(null);
  const [editingActivityTitle, setEditingActivityTitle] =
    useState("");

  function openActivityModal() {
    setActivityTitleInput("");
    setActivityError("");
    setIsActivityModalOpen(true);
  }

  function startEditingActivity(activity: Activity) {
    setEditingActivityID(activity.id);
    setEditingActivityTitle(activity.title);
    setActivityManagerError("");
  }

  function cancelEditingActivity() {
    setEditingActivityID(null);
    setEditingActivityTitle("");
  }

  async function openActivityManager() {
    setIsActivityManagerOpen(true);
    setActivityManagerLoading(true);
    setActivityManagerError("");

    try {
      await refreshManagedActivities();
    } catch (error) {
      setActivityManagerError(
        error instanceof Error
          ? error.message
          : "Failed to load activities",
      );
    } finally {
      setActivityManagerLoading(false);
    }
  }

  function closeActivityManager() {
    if (
      activityManagerLoading ||
      activityManagerSavingID !== null
    ) {
      return;
    }

    setIsActivityManagerOpen(false);
    setIsArchiveOpen(false);

    cancelEditingActivity();
    setActivityManagerError("");
  }

  async function refreshManagedActivities() {
    const result = await getActivities(true);
    setManagedActivities(result);
  }

  async function setActivityArchived(
    activity: Activity,
    isArchived: boolean,
  ) {
    setActivityManagerSavingID(activity.id);
    setActivityManagerError("");

    try {
      await updateActivity(activity.id, {
        is_archived: isArchived,
      });

      await refreshManagedActivities();

      const activeActivities = await getActivities();
      setActivities(activeActivities);

      if (
        isArchived &&
        selectedActivityID === activity.id
      ) {
        setSelectedActivityID("");
      }
    } catch (error) {
      setActivityManagerError(
        error instanceof Error
          ? error.message
          : isArchived
            ? "Failed to archive activity"
            : "Failed to restore activity",
      );
    } finally {
      setActivityManagerSavingID(null);
    }
  }

  async function saveActivityTitle(activity: Activity) {
    const title = editingActivityTitle.trim();

    if (!title) {
      setActivityManagerError("Activity title is required");
      return;
    }

    if (title === activity.title) {
      cancelEditingActivity();
      return;
    }

    setActivityManagerSavingID(activity.id);
    setActivityManagerError("");

    try {
      await updateActivity(activity.id, {
        title,
      });

      await refreshManagedActivities();

      const activeActivities = await getActivities();
      setActivities(activeActivities);

      cancelEditingActivity();
    } catch (error) {
      setActivityManagerError(
        error instanceof Error
          ? error.message
          : "Failed to rename activity",
      );
    } finally {
      setActivityManagerSavingID(null);
    }
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
      setActivityError("Activity title is required");
      return;
    }

    setActivitySaving(true);
    setActivityError("");

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
      setActivityError(
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
            className="goalEditButton dashboardSettingsButton"
            onClick={() => void openActivityManager()}
            disabled={loading || activitySaving}
            aria-label="Open settings and activity manager"
            title="Settings"
          >
            <span className="dashboardSettingsIcon" aria-hidden="true">
              ✎
            </span>
          </button>

          {!hasActiveSession && (
            <div
              className="activityPicker"
              style={{ width: `${activitySelectWidth}px` }}
            >
              <label className="activitySelector activitySelectorAboveRing">
                <span className="activitySelectorLabel">Activity</span>

                <select
                  className="activitySelectorInput"
                  value={selectedActivityID}
                  onChange={(event) =>
                    setSelectedActivityID(event.target.value)
                  }
                  disabled={loading || activitySaving}
                >
                  <option value="">-</option>

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
                <span className="activityCreateIcon" aria-hidden="true">
                  +
                </span>
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

      {isActivityManagerOpen && (
        <div
          className="activityManagerOverlay"
          role="presentation"
          onMouseDown={closeActivityManager}
        >
          <div
            className="activityManagerModal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="activity-manager-title"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <div className="activityManagerHeader">
              <div>
                <span className="activityManagerEyebrow">
                  Management
                </span>

                <h2 id="activity-manager-title">
                  Activities
                </h2>
              </div>

              <button
                type="button"
                className="activityManagerCloseButton"
                onClick={closeActivityManager}
                disabled={
                  activityManagerLoading ||
                  activityManagerSavingID !== null
                }
                aria-label="Close activity manager"
              >
                ×
              </button>
            </div>

            {activityManagerLoading && (
              <p className="emptyState">
                Loading activities...
              </p>
            )}

            {activityManagerError && (
              <p className="error">
                {activityManagerError}
              </p>
            )}

            <div className="activityManagerQuickSettings">
              <label className="activityManagerQuickField">
                <span>Daily goal</span>

                <input
                  type="text"
                  value={goalInput}
                  onChange={(event) =>
                    setGoalInput(event.target.value.slice(0, 5))
                  }
                  maxLength={5}
                  placeholder="2h"
                  disabled={goalSaving}
                  aria-label="Daily goal"
                />
              </label>

              <label className="activityManagerQuickField">
                <span className="activityManagerQuickLabel">
                  Day start

                  <span
                    className="activityManagerInfo"
                    tabIndex={0}
                    aria-label="Sessions before this hour count toward the previous day"
                  >
                    i

                    <span
                      className="activityManagerInfoTooltip"
                      role="tooltip"
                    >
                      Choose the hour when a new day begins. Enter a value from 0 to 23; the default is 0 (midnight).
                    </span>
                  </span>
                </span>
                <div className="activityManagerQuickHour">
                  <input
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={dayStartHourInput}
                    onChange={(event) => {
                      const nextValue = event.target.value
                        .replace(/\D/g, "")
                        .slice(0, 2);

                      setDayStartHourInput(nextValue);
                    }}
                    maxLength={2}
                    placeholder="0"
                    disabled={goalSaving}
                    aria-label="Day start hour"
                  />

                  <span>:00</span>
                </div>
              </label>

              <button
                type="button"
                className="activityManagerQuickSave"
                onClick={() => void saveSettings()}
                disabled={
                  goalSaving ||
                  !goalInput.trim() ||
                  !dayStartHourInput.trim()
                }
              >
                {goalSaving ? "Saving..." : "Save"}
              </button>
            </div>

            {!activityManagerLoading &&
              !activityManagerError && (
                <div className="activityManagerColumns">
                  <section className="activityManagerSection">
                    <div className="activityManagerSectionHeader">
                      <h3>Active</h3>

                      <span>
                        {activeManagedActivities.length}
                      </span>
                    </div>

                    {activeManagedActivities.length === 0 ? (
                      <p className="emptyState">
                        No active activities.
                      </p>
                    ) : (
                      <div className="activityManagerList">
                        {activeManagedActivities.map(
                          (activity) => (
                            <div
                              className="activityManagerItem"
                              key={activity.id}
                            >
                              {editingActivityID ===
                                activity.id ? (
                                <>
                                  <input
                                    type="text"
                                    value={editingActivityTitle}
                                    onChange={(event) =>
                                      setEditingActivityTitle(
                                        event.target.value,
                                      )
                                    }
                                    onKeyDown={(event) => {
                                      if (event.key === "Enter") {
                                        void saveActivityTitle(
                                          activity,
                                        );
                                      }

                                      if (event.key === "Escape") {
                                        cancelEditingActivity();
                                      }
                                    }}
                                    disabled={
                                      activityManagerSavingID !==
                                      null
                                    }
                                    autoFocus
                                  />

                                  <div className="activityManagerActions">
                                    <button
                                      type="button"
                                      className="activityManagerActionButton activityManagerCancelButton"
                                      onClick={
                                        cancelEditingActivity
                                      }
                                      disabled={
                                        activityManagerSavingID !==
                                        null
                                      }
                                    >
                                      Cancel
                                    </button>


                                    <button
                                      type="button"
                                      className="activityManagerActionButton activityManagerSaveButton"
                                      onClick={() =>
                                        void saveActivityTitle(
                                          activity,
                                        )
                                      }
                                      disabled={
                                        activityManagerSavingID !==
                                        null ||
                                        editingActivityTitle.trim()
                                          .length === 0
                                      }
                                    >
                                      {activityManagerSavingID ===
                                        activity.id
                                        ? "Saving..."
                                        : "Save"}
                                    </button>
                                  </div>
                                </>
                              ) : (
                                <>
                                  <span
                                    className="activityManagerTitle"
                                    title={activity.title}
                                  >
                                    {activity.title}
                                  </span>

                                  <div className="activityManagerActions">
                                    <button
                                      type="button"
                                      className="activityManagerActionButton"
                                      onClick={() =>
                                        startEditingActivity(
                                          activity,
                                        )
                                      }
                                      disabled={
                                        activityManagerSavingID !==
                                        null
                                      }
                                    >
                                      Rename
                                    </button>

                                    <button
                                      type="button"
                                      className="activityManagerActionButton activityManagerArchiveButton"
                                      onClick={() =>
                                        void setActivityArchived(
                                          activity,
                                          true,
                                        )
                                      }
                                      disabled={
                                        activityManagerSavingID !==
                                        null
                                      }
                                    >
                                      {activityManagerSavingID ===
                                        activity.id
                                        ? "Archiving..."
                                        : "Archive"}
                                    </button>
                                  </div>
                                </>
                              )}
                            </div>
                          ),
                        )}
                      </div>
                    )}
                  </section>

                  <section
                    className={
                      isArchiveOpen
                        ? "activityManagerSection activityManagerArchivedSection activityManagerArchivedSectionOpen"
                        : "activityManagerSection activityManagerArchivedSection"
                    }
                  >
                    <button
                      type="button"
                      className="activityManagerArchiveToggle"
                      onClick={() =>
                        setIsArchiveOpen((currentValue) => !currentValue)
                      }
                      aria-expanded={isArchiveOpen}
                      aria-controls="archived-activities-list"
                    >
                      <span className="activityManagerArchiveToggleTitle">
                        <span
                          className={
                            isArchiveOpen
                              ? "activityManagerArchiveChevron activityManagerArchiveChevronOpen"
                              : "activityManagerArchiveChevron"
                          }
                          aria-hidden="true"
                        >
                          ›
                        </span>

                        Archived
                      </span>

                      <span className="activityManagerArchiveCount">
                        {archivedManagedActivities.length}
                      </span>
                    </button>

                    {isArchiveOpen && (
                      <div
                        className="activityManagerArchiveContent"
                        id="archived-activities-list"
                      >
                        {archivedManagedActivities.length === 0 ? (
                          <p className="emptyState">
                            No archived activities.
                          </p>
                        ) : (
                          <div className="activityManagerList">
                            {archivedManagedActivities.map((activity) => (
                              <div
                                className="activityManagerItem"
                                key={activity.id}
                              >
                                {editingActivityID === activity.id ? (
                                  <>
                                    <input
                                      type="text"
                                      value={editingActivityTitle}
                                      onChange={(event) =>
                                        setEditingActivityTitle(
                                          event.target.value,
                                        )
                                      }
                                      onKeyDown={(event) => {
                                        if (event.key === "Enter") {
                                          void saveActivityTitle(activity);
                                        }

                                        if (event.key === "Escape") {
                                          cancelEditingActivity();
                                        }
                                      }}
                                      disabled={
                                        activityManagerSavingID !== null
                                      }
                                      autoFocus
                                    />

                                    <div className="activityManagerActions">
                                      <button
                                        type="button"
                                        className="activityManagerActionButton activityManagerCancelButton"
                                        onClick={cancelEditingActivity}
                                        disabled={
                                          activityManagerSavingID !== null
                                        }
                                      >
                                        Cancel
                                      </button>


                                      <button
                                        type="button"
                                        className="activityManagerActionButton activityManagerSaveButton"
                                        onClick={() =>
                                          void saveActivityTitle(activity)
                                        }
                                        disabled={
                                          activityManagerSavingID !== null ||
                                          editingActivityTitle.trim().length === 0
                                        }
                                      >
                                        {activityManagerSavingID === activity.id
                                          ? "Saving..."
                                          : "Save"}
                                      </button>

                                    </div>
                                  </>
                                ) : (
                                  <>
                                    <span
                                      className="activityManagerTitle"
                                      title={activity.title}
                                    >
                                      {activity.title}
                                    </span>

                                    <div className="activityManagerActions">
                                      <button
                                        type="button"
                                        className="activityManagerActionButton"
                                        onClick={() =>
                                          startEditingActivity(activity)
                                        }
                                        disabled={
                                          activityManagerSavingID !== null
                                        }
                                      >
                                        Rename
                                      </button>

                                      <button
                                        type="button"
                                        className="activityManagerActionButton activityManagerRestoreButton"
                                        onClick={() =>
                                          void setActivityArchived(
                                            activity,
                                            false,
                                          )
                                        }
                                        disabled={
                                          activityManagerSavingID !== null
                                        }
                                      >
                                        {activityManagerSavingID === activity.id
                                          ? "Restoring..."
                                          : "Restore"}
                                      </button>
                                    </div>
                                  </>
                                )}
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </section>
                </div>
              )}
          </div>
        </div>
      )}

      {isGoalModalOpen && (
        <div
          className="activityManagerOverlay"
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
          className="activityManagerOverlay"
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
                onChange={(event) => {
                  setActivityTitleInput(event.target.value);

                  if (activityError) {
                    setActivityError("");
                  }
                }}
                placeholder="For example: Programming"
                disabled={activitySaving}
                aria-label="Activity title"
                autoFocus
              />
            </label>

            {activityError && (
              <p className="error activityModalError">
                {activityError}
              </p>
            )}

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