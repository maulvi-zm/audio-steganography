import React, { useRef } from "react";
import MediaPlayer from "./ui/MediaPlayer";

interface FileUploadProps {
	label: string;
	accept: string;
	file: File | null;
	onFileChange: (file: File | null) => void;
	required?: boolean;
	className?: string;
}

const FileUpload: React.FC<FileUploadProps> = ({
	label,
	accept,
	file,
	onFileChange,
	required = false,
	className = "",
}) => {
	const fileInputRef = useRef<HTMLInputElement>(null);

	const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
		const selectedFile = event.target.files?.[0] || null;
		onFileChange(selectedFile);
	};

	const handleClick = () => {
		fileInputRef.current?.click();
	};

	const handleDrop = (event: React.DragEvent<HTMLDivElement>) => {
		event.preventDefault();
		const droppedFile = event.dataTransfer.files?.[0] || null;
		onFileChange(droppedFile);
	};

	const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
		event.preventDefault();
	};

	const isAudioFile = file && file.type.includes("audio");

	return (
		<div className={`mb-4 ${className}`}>
			<label className="block text-sm font-medium text-zinc-300 mb-2">
				{label} {required && <span className="text-red-500">*</span>}
			</label>

			<div
				className="border-2 border-dashed border-zinc-600 bg-zinc-900 rounded-lg p-6 text-center cursor-pointer hover:border-zinc-500 transition-colors"
				onClick={handleClick}
				onDrop={handleDrop}
				onDragOver={handleDragOver}
			>
				<input
					ref={fileInputRef}
					type="file"
					accept={accept}
					onChange={handleFileChange}
					className="hidden"
					required={required}
				/>

				{file ? (
					<div className="text-green-400">
						<svg
							className="mx-auto h-12 w-12 mb-2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<p className="text-sm font-medium text-white">{file.name}</p>
						<p className="text-xs text-zinc-400">
							{(file.size / 1024 / 1024).toFixed(2)} MB
						</p>
					</div>
				) : (
					<div className="text-zinc-400">
						<svg
							className="mx-auto h-12 w-12 mb-2"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
							/>
						</svg>
						<p className="text-sm text-zinc-300">
							Click to select or drag and drop
						</p>
						<p className="text-xs text-zinc-500">{accept}</p>
					</div>
				)}
			</div>

			{/* Media Player for audio files */}
			{isAudioFile && (
				<div className="mt-3">
					<MediaPlayer file={file} title="Preview" />
				</div>
			)}
		</div>
	);
};

export default FileUpload;
