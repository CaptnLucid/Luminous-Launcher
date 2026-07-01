// frontend/src/App.tsx
import React, { useEffect, useState } from "react";
import {
  CheckLauncherUpdates,
  LoadAvailableProfiles,
  ExecuteGame,
  ApplyApplicationUpdate,
  ApplyNipProfile,
  GetCurrentVersion,
  SaveSettingUpdate,
  GetSettings,
  GetRecommendedAffinity,
  MinimizeWindow,
  ToggleMaximizeWindow,
  IsWindowMaximized,
  QuitLayout,
} from "../wailsjs/go/backend/App";
import appIcon from "./assets/images/appicon.png";
import "./App.css";

interface UpdateData {
  has_update: boolean;
  version: string;
  changelog: string;
  url: string;
  download_url: string;
}

interface RecommendedAffinity {
  cpuName: string;
  matched: boolean;
  label: string;
  affinityMask: string;
  enable: string;
  note: string;
}

// GitHub tags are usually prefixed with "v" (e.g. "v1.4.0"); the backend's
// CurrentVersion constant is not. Normalize both before display so the
// header always shows a consistent "1.4.0"-style version string regardless
// of which source it came from.
const normalizeVersion = (v: string) => v.replace(/^v/i, "");

const DEFAULT_AFFINITY = "FFFF";

export default function App() {
  const [profiles, setProfiles] = useState<Record<string, string>>({});
  const [selectedProfile, setSelectedProfile] = useState("");
  const [launchMode, setLaunchMode] = useState("pearl");
  const [customPath, setCustomPath] = useState("");
  const [affinity, setAffinity] = useState("FFFF");
  const [consoleLogs, setConsoleLogs] = useState<string[]>([]);
  const [isLaunching, setIsLaunching] = useState(false);
  const [isApplyingProfile, setIsApplyingProfile] = useState(false);
  const [currentVersion, setCurrentVersion] = useState("1.0.0");
  const [executablePath, setExecutablePath] = useState("");
  const [isSteam, setIsSteam] = useState(false);
  const [affinityMask, setAffinityMask] = useState("0");

  const [remoteUpdate, setRemoteUpdate] = useState<UpdateData | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);

  const [knownCPU, setKnownCPU] = useState<RecommendedAffinity | null>(null);
  const [isDetectingKnownCPU, setIsDetectingKnownCPU] = useState(false);

  const [isMaximized, setIsMaximized] = useState(false);

  useEffect(() => {
    LoadAvailableProfiles().then((res) => {
      setProfiles(res || {});
      addLog("info", "System profile paths scanned.");
    });

    IsWindowMaximized()
      .then((maximized: boolean) => setIsMaximized(maximized))
      .catch(() => {});

    GetCurrentVersion()
      .then((v: string) => {
        setCurrentVersion(v);
      })
      .catch((err) => {
        console.error(err);
      });

    GetSettings()
      .then((config) => {
        if (config) {
          setExecutablePath(config.executablePath);
          setCustomPath(config.executablePath);
          setIsSteam(config.isSteam);

          const savedMask = String(config.affinityMask || "0");
          setAffinityMask(savedMask);

          if (savedMask !== "0" && savedMask !== "") {
            setAffinity(savedMask.toUpperCase());
          }

          if (config.selectedProfile) {
            setSelectedProfile(config.selectedProfile);
          }

          // The backend now persists the Distribution Target selection
          // directly. Fall back to the old path/steam heuristic only for
          // config files saved before that field existed.
          if (config.launchMode) {
            setLaunchMode(config.launchMode);
          } else if (config.executablePath) {
            setLaunchMode("custom");
          } else if (config.isSteam) {
            setLaunchMode("steam");
          }
        }
      })
      .catch((err) => console.error("Failed to load settings:", err));

    CheckLauncherUpdates()
      .then((updateInfo) => {
        if (updateInfo && updateInfo.has_update) {
          setRemoteUpdate(updateInfo);
          addLog("warn", `Update available — ${updateInfo.version}`);
        } else {
          addLog("info", "Launcher is up to date.");
        }
      })
      .catch((err) => addLog("error", `Update check failed: ${err}`));
  }, []);

  const handleSettingsChange = (
    mode: string,
    path: string,
    steamToggle: boolean,
    maskStr: string,
    profile: string,
  ) => {
    setLaunchMode(mode);
    setExecutablePath(path);
    setCustomPath(path);
    setIsSteam(steamToggle);
    setAffinityMask(maskStr);
    SaveSettingUpdate(mode, path, steamToggle, maskStr, profile);
  };

  const addLog = (type: "info" | "warn" | "error" | "ok", message: string) => {
    const timestamp = new Date().toLocaleTimeString("en-US", {
      hour12: false,
      hour: "2-digit",
      minute: "2-digit",
    });
    const entryString = `${type}|${timestamp}|${message}`;

    setConsoleLogs((prev) => {
      if (prev.length > 0) {
        const lastEntry = prev[prev.length - 1];
        const [, , lastMsg] = lastEntry.split("|");
        if (lastMsg === message) {
          return prev;
        }
      }
      return [...prev, entryString];
    });
  };

  const handleLaunch = async () => {
    setIsLaunching(true);
    addLog("info", "Initializing launch sequence...");

    const finalMode = launchMode === "custom" && isSteam ? "steam" : launchMode;
    const finalPath = launchMode === "custom" ? executablePath : customPath;
    const finalAffinity =
      affinityMask !== "0" && affinityMask !== "" ? affinityMask : affinity;

    const result = await ExecuteGame(
      finalMode,
      finalPath,
      finalAffinity.toUpperCase(),
    );
    addLog("ok", `${result}`);
    setIsLaunching(false);
  };

  const handleApplyProfile = async () => {
    if (!selectedProfile) {
      addLog("warn", "No profile selected.");
      return;
    }
    const nipPath = profiles[selectedProfile] || "";
    setIsApplyingProfile(true);
    addLog("info", `Injecting profile: ${selectedProfile}.nip`);
    const result = await ApplyNipProfile(nipPath);
    if (result === "Success") {
      addLog("ok", `Profile applied: ${selectedProfile}`);
    } else {
      addLog("error", result);
    }
    setIsApplyingProfile(false);
  };

  const handleSystemUpdate = async () => {
    if (!remoteUpdate) return;
    setIsUpdating(true);
    addLog("info", "Downloading new executable engine layer from GitHub...");

    const statusResult = await ApplyApplicationUpdate(
      remoteUpdate.download_url,
    );

    if (statusResult !== "Success") {
      addLog("error", `Update sequence aborted: ${statusResult}`);
      setIsUpdating(false);
    }
  };

  const handleMinimize = () => {
    MinimizeWindow();
  };

  const handleToggleMaximize = () => {
    ToggleMaximizeWindow();
    // Optimistic flip; corrected on next mount/IsWindowMaximized check if
    // the OS declines the request for some reason (e.g. a fixed-size window).
    setIsMaximized((prev) => !prev);
  };

  const handleCloseWindow = () => {
    QuitLayout();
  };

  // Looks up the installed CPU against the hardcoded Ryzen affinity table and
  // surfaces the community-recommended mask, or FFFF if the CPU isn't a
  // recognized match.
  const handleDetectKnownCPU = async () => {
    setIsDetectingKnownCPU(true);
    addLog("info", "Checking installed CPU against known affinity profiles...");
    try {
      const result = (await GetRecommendedAffinity()) as RecommendedAffinity;
      setKnownCPU(result);

      if (result.matched) {
        addLog(
          "ok",
          `Recognized ${result.label} — recommended mask 0x${result.affinityMask}`,
        );
      } else {
        addLog(
          "warn",
          `CPU "${result.cpuName || "unknown"}" not in the known profile database — using default 0x${DEFAULT_AFFINITY}`,
        );
      }
    } catch (err) {
      addLog("error", "Failed to read CPU information from the system.");
    }
    setIsDetectingKnownCPU(false);
  };

  const logColor: Record<string, string> = {
    info: "var(--text-secondary)",
    warn: "var(--warn)",
    error: "var(--error)",
    ok: "var(--ok)",
  };

  return (
    <div className="app">
      <div className="titlebar" onDoubleClick={handleToggleMaximize}>
        <div className="titlebar-drag-area">
          <img src={appIcon} alt="" className="titlebar-icon" />
          <span className="titlebar-label">LUMINOUS LAUNCHER</span>
        </div>

        <div className="titlebar-controls">
          <button
            className="titlebar-btn"
            onClick={handleMinimize}
            aria-label="Minimize"
            title="Minimize"
          >
            <svg viewBox="0 0 10 10" width="10" height="10">
              <rect x="0" y="4.5" width="10" height="1" fill="currentColor" />
            </svg>
          </button>

          <button
            className="titlebar-btn"
            onClick={handleToggleMaximize}
            aria-label={isMaximized ? "Restore" : "Maximize"}
            title={isMaximized ? "Restore" : "Maximize"}
          >
            {isMaximized ? (
              <svg viewBox="0 0 10 10" width="10" height="10">
                <rect
                  x="1.5"
                  y="0.5"
                  width="7"
                  height="7"
                  fill="none"
                  stroke="currentColor"
                />
                <rect
                  x="0.5"
                  y="2.5"
                  width="7"
                  height="7"
                  fill="var(--bg-elevated)"
                  stroke="currentColor"
                />
              </svg>
            ) : (
              <svg viewBox="0 0 10 10" width="10" height="10">
                <rect
                  x="0.5"
                  y="0.5"
                  width="9"
                  height="9"
                  fill="none"
                  stroke="currentColor"
                />
              </svg>
            )}
          </button>

          <button
            className="titlebar-btn titlebar-btn-close"
            onClick={handleCloseWindow}
            aria-label="Close"
            title="Close"
          >
            <svg viewBox="0 0 10 10" width="10" height="10">
              <line x1="0.5" y1="0.5" x2="9.5" y2="9.5" stroke="currentColor" />
              <line x1="9.5" y1="0.5" x2="0.5" y2="9.5" stroke="currentColor" />
            </svg>
          </button>
        </div>
      </div>

      <header className="app-header">
        <img src={appIcon} alt="" className="logo-mark" />
        <h1 className="app-title">
          LUMINOUS LAUNCHER
          <span className="app-subtitle">
            Custom BDO Launcher v{normalizeVersion(currentVersion)}
          </span>
        </h1>

        <div className="header-actions">
          {!remoteUpdate && (
            <span className="version-pill" title="Launcher is up to date">
              v{normalizeVersion(currentVersion)}
            </span>
          )}

          {remoteUpdate && (
            <button
              onClick={handleSystemUpdate}
              disabled={isUpdating}
              className="btn btn-update"
            >
              {isUpdating
                ? "UPDATING..."
                : `UPDATE AVAILABLE — v${normalizeVersion(remoteUpdate.version)}`}
            </button>
          )}
        </div>
      </header>

      <main className="app-body">
        <div className="col col-left">
          <h2 className="section-title">Launch Configuration</h2>

          <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <label className="field-label">Distribution Target</label>
            <select
              className="select-input"
              value={launchMode}
              onChange={(e) =>
                handleSettingsChange(
                  e.target.value,
                  executablePath,
                  isSteam,
                  affinityMask,
                  selectedProfile,
                )
              }
            >
              <option value="pearl">Pearl Abyss Launcher (Standalone)</option>
              <option value="steam">Steam Client</option>
              <option value="custom">Custom Path</option>
            </select>
          </div>

          <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <label className="field-label">CPU Affinity Mask</label>
            <div style={{ display: "flex", gap: "8px", alignItems: "center" }}>
              <input
                type="text"
                className="text-input"
                style={{
                  fontFamily: "var(--font-mono)",
                  textTransform: "uppercase",
                }}
                value={affinity}
                disabled={affinityMask !== "0" && affinityMask !== ""}
                onChange={(e) => {
                  const uppercaseVal = e.target.value.toUpperCase();
                  setAffinity(uppercaseVal);
                  if (affinityMask !== "0" && affinityMask !== "") {
                    handleSettingsChange(
                      launchMode,
                      executablePath,
                      isSteam,
                      uppercaseVal,
                      selectedProfile,
                    );
                  }
                }}
                placeholder="FFFF"
              />

              <button
                className="btn btn-secondary"
                onClick={handleDetectKnownCPU}
                disabled={isDetectingKnownCPU}
                title="Look up a known-good mask for this CPU model"
              >
                {isDetectingKnownCPU ? "CHECKING..." : "KNOWN CPU"}
              </button>

              <label
                className="custom-checkbox-label"
                title="Lock Affinity Profile"
              >
                <input
                  type="checkbox"
                  className="custom-checkbox"
                  checked={affinityMask !== "0" && affinityMask !== ""}
                  onChange={(e) => {
                    const nextMask = e.target.checked ? affinity : "0";
                    handleSettingsChange(
                      launchMode,
                      executablePath,
                      isSteam,
                      nextMask,
                      selectedProfile,
                    );
                  }}
                />
                <span className="field-label">Lock</span>
              </label>
            </div>

            {knownCPU && (
              <div className="cpu-match-card">
                {knownCPU.matched ? (
                  <>
                    <span className="cpu-match-title">{knownCPU.label}</span>
                    <span className="cpu-match-detail">
                      Recommended mask <code>0x{knownCPU.affinityMask}</code>
                      {knownCPU.enable
                        ? ` — enable cores ${knownCPU.enable}`
                        : ""}
                    </span>
                    {knownCPU.note && (
                      <span className="cpu-match-note">{knownCPU.note}</span>
                    )}
                  </>
                ) : (
                  <span className="cpu-match-detail">
                    {knownCPU.cpuName
                      ? `"${knownCPU.cpuName}" isn't in the known profile database`
                      : "Couldn't identify the installed CPU"}{" "}
                    — defaulting to <code>0x{DEFAULT_AFFINITY}</code>
                  </span>
                )}
              </div>
            )}
          </div>

          {launchMode === "custom" && (
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                gap: "12px",
                padding: "4px 0",
              }}
            >
              <div
                style={{ display: "flex", flexDirection: "column", gap: "6px" }}
              >
                <label className="field-label">Executable Path Override</label>
                <input
                  type="text"
                  className="text-input"
                  value={executablePath}
                  onChange={(e) =>
                    handleSettingsChange(
                      launchMode,
                      e.target.value,
                      isSteam,
                      affinityMask,
                      selectedProfile,
                    )
                  }
                  placeholder="D:\BlackDesert\BlackDesertLauncher.exe"
                />
              </div>

              {/* Redesigned Custom Checkbox Component Layout Row */}
              <div style={{ display: "flex", gap: "20px", marginTop: "4px" }}>
                <label className="custom-checkbox-label">
                  <input
                    type="checkbox"
                    className="custom-checkbox"
                    checked={!isSteam}
                    onChange={() =>
                      handleSettingsChange(
                        launchMode,
                        executablePath,
                        false,
                        affinityMask,
                        selectedProfile,
                      )
                    }
                  />
                  <span className="field-label">Standalone Web Client</span>
                </label>

                <label className="custom-checkbox-label">
                  <input
                    type="checkbox"
                    className="custom-checkbox"
                    checked={isSteam}
                    onChange={() =>
                      handleSettingsChange(
                        launchMode,
                        executablePath,
                        true,
                        affinityMask,
                        selectedProfile,
                      )
                    }
                  />
                  <span className="field-label">Steam Client</span>
                </label>
              </div>
            </div>
          )}

          <hr className="divider" />

          <div style={{ display: "flex", flexDirection: "column", gap: "6px" }}>
            <label className="field-label">
              Nvidia Inspector Profile (.nip)
            </label>
            <div style={{ display: "flex", gap: "8px" }}>
              <select
                className="select-input"
                value={selectedProfile}
                onChange={(e) => {
                  const newProfile = e.target.value;
                  setSelectedProfile(newProfile);
                  handleSettingsChange(
                    launchMode,
                    executablePath,
                    isSteam,
                    affinityMask,
                    newProfile,
                  );
                }}
              >
                <option value="">No profile — skip injection</option>
                {Object.keys(profiles).map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </select>

              {selectedProfile && (
                <button
                  className="btn btn-secondary"
                  onClick={handleApplyProfile}
                  disabled={isApplyingProfile}
                >
                  {isApplyingProfile ? "APPLYING..." : "APPLY"}
                </button>
              )}
            </div>
          </div>
        </div>

        <div className="col col-right">
          <h2 className="section-title">Runtime Log</h2>

          <div
            className="log-viewer"
            style={{
              background: "var(--surface)",
              border: "1px solid var(--border)",
              borderRadius: "6px",
              padding: "12px",
              fontFamily: "var(--font-mono)",
              fontSize: "11px",
              overflowY: "auto",
              display: "flex",
              flexDirection: "column",
              gap: "8px",
            }}
          >
            {consoleLogs.length === 0 && (
              <span style={{ color: "var(--text-muted)" }}>
                Awaiting activity...
              </span>
            )}
            {consoleLogs.map((entry, i) => {
              const [type, time, ...rest] = entry.split("|");
              return (
                <div
                  key={i}
                  style={{
                    display: "flex",
                    gap: "10px",
                    alignItems: "flex-start",
                    wordBreak: "break-all",
                  }}
                >
                  <span style={{ color: "var(--text-muted)", flexShrink: 0 }}>
                    [{time}]
                  </span>
                  <span
                    style={{
                      color: logColor[type] || "var(--text-secondary)",
                      flex: 1,
                      whiteSpace: "pre-wrap",
                    }}
                  >
                    {rest.join("|")}
                  </span>
                </div>
              );
            })}
          </div>

          <button
            className="btn btn-launch"
            onClick={handleLaunch}
            disabled={isLaunching}
          >
            {isLaunching ? (
              <>
                <span className="spinner" /> LAUNCHING...
              </>
            ) : (
              "LAUNCH GAME"
            )}
          </button>
        </div>
      </main>
    </div>
  );
}
