import React, { useRef, useState, useEffect } from "react";
import { Play, Pause, Volume2, VolumeX } from "lucide-react";
import { cn } from "../../lib/utils";

interface MediaPlayerProps {
	file: File | null;
	className?: string;
	title?: string;
}

const MediaPlayer: React.FC<MediaPlayerProps> = ({
	file,
	className,
	title,
}) => {
	const audioRef = useRef<HTMLAudioElement>(null);
	const [isPlaying, setIsPlaying] = useState(false);
	const [currentTime, setCurrentTime] = useState(0);
	const [duration, setDuration] = useState(0);
	const [volume, setVolume] = useState(1);
	const [isMuted, setIsMuted] = useState(false);
	const [audioUrl, setAudioUrl] = useState<string | null>(null);

	useEffect(() => {
		if (file) {
			const url = URL.createObjectURL(file);
			setAudioUrl(url);

			return () => {
				URL.revokeObjectURL(url);
				setAudioUrl(null);
			};
		}
	}, [file]);

	useEffect(() => {
		const audio = audioRef.current;
		if (!audio) return;

		const updateTime = () => setCurrentTime(audio.currentTime);
		const updateDuration = () => setDuration(audio.duration);
		const handleEnd = () => setIsPlaying(false);

		audio.addEventListener("timeupdate", updateTime);
		audio.addEventListener("loadedmetadata", updateDuration);
		audio.addEventListener("ended", handleEnd);

		return () => {
			audio.removeEventListener("timeupdate", updateTime);
			audio.removeEventListener("loadedmetadata", updateDuration);
			audio.removeEventListener("ended", handleEnd);
		};
	}, [audioUrl]);

	const togglePlay = () => {
		const audio = audioRef.current;
		if (!audio) return;

		if (isPlaying) {
			audio.pause();
		} else {
			audio.play();
		}
		setIsPlaying(!isPlaying);
	};

	const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
		const audio = audioRef.current;
		if (!audio) return;

		const seekTime = (parseFloat(e.target.value) / 100) * duration;
		audio.currentTime = seekTime;
		setCurrentTime(seekTime);
	};

	const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
		const newVolume = parseFloat(e.target.value) / 100;
		setVolume(newVolume);
		if (audioRef.current) {
			audioRef.current.volume = newVolume;
		}
		setIsMuted(newVolume === 0);
	};

	const toggleMute = () => {
		const audio = audioRef.current;
		if (!audio) return;

		if (isMuted) {
			audio.volume = volume;
			setIsMuted(false);
		} else {
			audio.volume = 0;
			setIsMuted(true);
		}
	};

	const formatTime = (time: number) => {
		if (isNaN(time)) return "0:00";
		const minutes = Math.floor(time / 60);
		const seconds = Math.floor(time % 60);
		return `${minutes}:${seconds.toString().padStart(2, "0")}`;
	};

	if (!file || !audioUrl) {
		return null;
	}

	return (
		<div
			className={cn(
				"bg-zinc-900 border border-zinc-800 rounded-lg p-4",
				className,
			)}
		>
			<audio ref={audioRef} src={audioUrl} />

			{title && (
				<div className="mb-3">
					<h4 className="text-sm font-medium text-white truncate">{title}</h4>
					<p className="text-xs text-zinc-400 truncate">{file.name}</p>
				</div>
			)}

			<div className="space-y-3">
				{/* Progress Bar */}
				<div className="space-y-1">
					<input
						type="range"
						min="0"
						max="100"
						value={duration ? (currentTime / duration) * 100 : 0}
						onChange={handleSeek}
						className="w-full h-1 bg-zinc-700 rounded-lg appearance-none cursor-pointer progress-slider"
					/>
					<div className="flex justify-between text-xs text-zinc-400">
						<span>{formatTime(currentTime)}</span>
						<span>{formatTime(duration)}</span>
					</div>
				</div>

				{/* Controls */}
				<div className="flex items-center justify-between">
					<div className="flex items-center space-x-3">
						<button
							onClick={togglePlay}
							className="flex items-center justify-center w-8 h-8 bg-white text-black rounded-full hover:bg-zinc-200 transition-colors"
						>
							{isPlaying ? (
								<Pause className="w-4 h-4" />
							) : (
								<Play className="w-4 h-4 ml-0.5" />
							)}
						</button>
					</div>

					<div className="flex items-center space-x-2">
						<button
							onClick={toggleMute}
							className="text-zinc-400 hover:text-white transition-colors"
						>
							{isMuted ? (
								<VolumeX className="w-4 h-4" />
							) : (
								<Volume2 className="w-4 h-4" />
							)}
						</button>
						<input
							type="range"
							min="0"
							max="100"
							value={isMuted ? 0 : volume * 100}
							onChange={handleVolumeChange}
							className="w-16 h-1 bg-zinc-700 rounded-lg appearance-none cursor-pointer volume-slider"
						/>
					</div>
				</div>
			</div>
		</div>
	);
};

export default MediaPlayer;
