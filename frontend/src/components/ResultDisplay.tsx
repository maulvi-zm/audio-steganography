import React, { useState, useEffect } from "react";
import MediaPlayer from "./ui/MediaPlayer";

interface ResultDisplayProps {
	success: boolean;
	message: string;
	psnr?: number;
	downloadUrl?: string;
	filename?: string;
	onReset: () => void;
}

const ResultDisplay: React.FC<ResultDisplayProps> = ({
	success,
	message,
	psnr,
	downloadUrl,
	filename,
	onReset,
}) => {
	const [audioFile, setAudioFile] = useState<File | null>(null);

	const handleDownload = () => {
		if (downloadUrl) {
			const link = document.createElement("a");
			link.href = downloadUrl;
			link.download = filename || "download";
			document.body.appendChild(link);
			link.click();
			document.body.removeChild(link);
		}
	};

	// Convert downloadUrl to File for audio preview
	useEffect(() => {
		if (downloadUrl && filename && filename.includes(".mp3")) {
			fetch(downloadUrl)
				.then((response) => response.blob())
				.then((blob) => {
					const file = new File([blob], filename, { type: "audio/mpeg" });
					setAudioFile(file);
				})
				.catch((error) => {
					console.error("Error creating audio file:", error);
				});
		} else {
			setAudioFile(null);
		}
	}, [downloadUrl, filename]);

	return (
		<div className="mt-6 p-4 bg-zinc-900 border border-zinc-700 rounded-lg">
			<div className="flex items-start space-x-3">
				<div className="flex-shrink-0">
					{success ? (
						<div className="w-8 h-8 bg-green-900 rounded-full flex items-center justify-center">
							<svg
								className="w-5 h-5 text-green-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M5 13l4 4L19 7"
								/>
							</svg>
						</div>
					) : (
						<div className="w-8 h-8 bg-red-900 rounded-full flex items-center justify-center">
							<svg
								className="w-5 h-5 text-red-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</div>
					)}
				</div>

				<div className="flex-1">
					<h3
						className={`text-lg font-medium ${success ? "text-green-400" : "text-red-400"}`}
					>
						{success ? "Success!" : "Error"}
					</h3>
					<p
						className={`mt-1 text-sm ${success ? "text-green-300" : "text-red-300"}`}
					>
						{message}
					</p>

					{success && psnr !== undefined && (
						<div className="mt-2 p-3 bg-zinc-800 border border-zinc-600 rounded-md">
							<p className="text-sm text-zinc-300">
								<strong>PSNR:</strong> {psnr.toFixed(2)} dB
								{psnr > 50 ? (
									<span className="ml-2 text-green-400">
										🟢 Excellent - Virtually indistinguishable
									</span>
								) : psnr >= 40 ? (
									<span className="ml-2 text-green-400">
										🟡 Good - High quality, minimal artifacts
									</span>
								) : psnr >= 30 ? (
									<span className="ml-2 text-yellow-400">
										🟠 Fair - Acceptable quality
									</span>
								) : (
									<span className="ml-2 text-red-400">
										🔴 Poor - Noticeable degradation
									</span>
								)}
							</p>
						</div>
					)}

					{/* Audio Preview */}
					{success && audioFile && (
						<div className="mt-3">
							<MediaPlayer file={audioFile} title="Result Audio" />
						</div>
					)}

					{success && downloadUrl && (
						<div className="mt-3 space-y-2">
							<button
								onClick={handleDownload}
								className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-black bg-white hover:bg-zinc-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-400"
							>
								<svg
									className="w-4 h-4 mr-2"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
									/>
								</svg>
								Download {filename ? filename : "File"}
							</button>
						</div>
					)}
				</div>
			</div>

			<div className="mt-4 flex justify-end">
				<button
					onClick={onReset}
					className="px-4 py-2 text-sm font-medium text-zinc-300 bg-zinc-800 border border-zinc-600 rounded-md hover:bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-400"
				>
					Start Over
				</button>
			</div>
		</div>
	);
};

export default ResultDisplay;
