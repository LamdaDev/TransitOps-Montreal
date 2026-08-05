import { Pause, Play, Radio, RotateCcw } from "lucide-react";
import type { RouteReplay } from "../types/transit";
import { formatDateTime } from "../utils/format";

interface ReplayControlsProps {
  error: string | null;
  frameIndex: number;
  isLoading: boolean;
  isPlaying: boolean;
  isReplayMode: boolean;
  replay: RouteReplay | null;
  speed: number;
  onEnterReplay: () => void;
  onExitReplay: () => void;
  onFrameIndexChange: (frameIndex: number) => void;
  onSpeedChange: (speed: number) => void;
  onTogglePlayback: () => void;
}

export function ReplayControls({
  error,
  frameIndex,
  isLoading,
  isPlaying,
  isReplayMode,
  replay,
  speed,
  onEnterReplay,
  onExitReplay,
  onFrameIndexChange,
  onSpeedChange,
  onTogglePlayback
}: ReplayControlsProps) {
  const frameCount = replay?.frames.length ?? 0;
  const currentFrame = replay?.frames[frameIndex];
  const hasFrames = frameCount > 0 && replay?.frames.some((frame) => frame.hasData);

  return (
    <section className="replay-panel" aria-label="Historical replay controls">
      <div className="replay-heading">
        <div>
          <span className="section-kicker">Historical playback</span>
          <h2>Vehicle replay</h2>
        </div>
        {isReplayMode ? (
          <button className="replay-live-button" onClick={onExitReplay} type="button">
            <Radio size={16} aria-hidden="true" />
            Return to live
          </button>
        ) : (
          <button className="replay-start-button" onClick={onEnterReplay} type="button">
            <Play size={16} aria-hidden="true" />
            Open replay
          </button>
        )}
      </div>

      {!isReplayMode ? (
        <p className="replay-description">
          Reconstruct stored vehicle positions at 15-second intervals without interrupting live updates.
        </p>
      ) : null}
      {isReplayMode && isLoading ? <div className="replay-state">Loading replay frames...</div> : null}
      {isReplayMode && error ? <div className="replay-state replay-state-error">{error}</div> : null}
      {isReplayMode && !isLoading && !error && !hasFrames ? (
        <div className="replay-state">No replayable snapshots are available for this range.</div>
      ) : null}

      {isReplayMode && !isLoading && !error && hasFrames && replay && currentFrame ? (
        <div className="replay-controls">
          <div className="replay-timestamp">
            <span>Replay state</span>
            <strong>{formatDateTime(currentFrame.timestamp)}</strong>
          </div>
          <div className="replay-actions">
            <button
              aria-label={isPlaying ? "Pause replay" : "Play replay"}
              className="replay-play-button"
              onClick={onTogglePlayback}
              type="button"
            >
              {isPlaying ? <Pause size={18} aria-hidden="true" /> : <Play size={18} aria-hidden="true" />}
              <span>{isPlaying ? "Pause" : "Play"}</span>
            </button>
            <button
              aria-label="Restart replay"
              className="replay-icon-button"
              onClick={() => onFrameIndexChange(0)}
              type="button"
            >
              <RotateCcw size={18} aria-hidden="true" />
            </button>
            <label className="replay-speed">
              <span>Speed</span>
              <select value={speed} onChange={(event) => onSpeedChange(Number(event.target.value))}>
                <option value={1}>1x</option>
                <option value={2}>2x</option>
                <option value={4}>4x</option>
              </select>
            </label>
          </div>
          <label className="replay-slider">
            <span>Position</span>
            <input
              aria-label="Replay position"
              max={Math.max(0, frameCount - 1)}
              min="0"
              onChange={(event) => onFrameIndexChange(Number(event.target.value))}
              type="range"
              value={frameIndex}
            />
            <span>{frameIndex + 1} / {frameCount}</span>
          </label>
        </div>
      ) : null}
    </section>
  );
}
