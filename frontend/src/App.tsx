import { useState } from "react";
import FileUpload from "./components/FileUpload";
import ConfigForm from "./components/ConfigForm";
import ResultDisplay from "./components/ResultDisplay";
import { steganographyAPI } from "./services/api";
import { StegoResponse, ExtractResponse } from "./types/api";

type Operation = "insert" | "extract";

function App() {
	const [operation, setOperation] = useState<Operation>("insert");
	const [loading, setLoading] = useState(false);
	const [result, setResult] = useState<{
		success: boolean;
		message: string;
		psnr?: number;
		downloadUrl?: string;
		filename?: string;
	} | null>(null);

	// Form state
	const [audioFile, setAudioFile] = useState<File | null>(null);
	const [secretFile, setSecretFile] = useState<File | null>(null);
	const [stegoAudioFile, setStegoAudioFile] = useState<File | null>(null);
	const [key, setKey] = useState("");
	const [useEncryption, setUseEncryption] = useState(false);
	const [useRandomStart, setUseRandomStart] = useState(false);
	const [lsbBits, setLsbBits] = useState(1);

	const resetForm = () => {
		// Clean up any existing blob URLs
		if (result?.downloadUrl) {
			steganographyAPI.cleanupBlobUrl(result.downloadUrl);
		}

		setAudioFile(null);
		setSecretFile(null);
		setStegoAudioFile(null);
		setKey("");
		setUseEncryption(false);
		setUseRandomStart(false);
		setLsbBits(1);
		setResult(null);
	};

	const handleInsert = async () => {
		if (!audioFile || !secretFile || !key) {
			alert("Please fill all required fields");
			return;
		}

		setLoading(true);
		try {
			const response: StegoResponse = await steganographyAPI.insertMessage(
				audioFile,
				secretFile,
				key,
				useEncryption,
				useRandomStart,
				lsbBits,
			);

			setResult({
				success: response.success,
				message: response.message,
				psnr: response.psnr,
				downloadUrl: response.download_url,
				filename: response.filename,
			});
		} catch (error: any) {
			setResult({
				success: false,
				message:
					error.response?.data?.message || error.message || "An error occurred",
			});
		} finally {
			setLoading(false);
		}
	};

	const handleExtract = async () => {
		if (!stegoAudioFile || !key) {
			alert("Please fill all required fields");
			return;
		}

		setLoading(true);
		try {
			const response: ExtractResponse = await steganographyAPI.extractMessage(
				stegoAudioFile,
				key,
				useEncryption,
				useRandomStart,
				lsbBits,
			);

			setResult({
				success: response.success,
				message: response.message,
				downloadUrl: response.download_url,
				filename: response.filename,
			});
		} catch (error: any) {
			setResult({
				success: false,
				message:
					error.response?.data?.message || error.message || "An error occurred",
			});
		} finally {
			setLoading(false);
		}
	};

	return (
		<div className="min-h-screen bg-black py-8">
			<div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
				<div className="bg-zinc-950 border border-zinc-800 shadow-xl rounded-lg overflow-hidden">
					{/* Header */}
					<div className="bg-zinc-900 border-b border-zinc-800 px-6 py-8">
						<h1 className="text-3xl font-bold text-white text-center">
							Audio Steganography Tool
						</h1>
						<p className="text-zinc-400 text-center mt-2">
							Hide and extract secret messages in MP3 audio files using
							Multiple-LSB method
						</p>
					</div>

					<div className="p-6 bg-zinc-950">
						{/* Operation Selector */}
						<div className="flex space-x-1 bg-zinc-900 border border-zinc-800 rounded-lg p-1 mb-6">
							<button
								onClick={() => setOperation("insert")}
								className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
									operation === "insert"
										? "bg-white text-black shadow-sm"
										: "text-zinc-400 hover:text-zinc-300"
								}`}
							>
								Insert Message
							</button>
							<button
								onClick={() => setOperation("extract")}
								className={`flex-1 py-2 px-4 rounded-md text-sm font-medium transition-colors ${
									operation === "extract"
										? "bg-white text-black shadow-sm"
										: "text-zinc-400 hover:text-zinc-300"
								}`}
							>
								Extract Message
							</button>
						</div>

						{/* Form Content */}
						<div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
							{/* File Uploads */}
							<div>
								<h2 className="text-xl font-semibold text-white mb-4">
									{operation === "insert"
										? "Upload Files"
										: "Upload Stego Audio"}
								</h2>

								{operation === "insert" ? (
									<>
										<FileUpload
											label="Cover Audio (MP3)"
											accept=".mp3,audio/mpeg"
											file={audioFile}
											onFileChange={setAudioFile}
											required
										/>
										<FileUpload
											label="Secret File"
											accept="*/*"
											file={secretFile}
											onFileChange={setSecretFile}
											required
										/>
									</>
								) : (
									<FileUpload
										label="Stego Audio (MP3)"
										accept=".mp3,audio/mpeg"
										file={stegoAudioFile}
										onFileChange={setStegoAudioFile}
										required
									/>
								)}
							</div>

							{/* Configuration */}
							<div>
								<h2 className="text-xl font-semibold text-white mb-4">
									Configuration
								</h2>
								<ConfigForm
									stringKey={key}
									useEncryption={useEncryption}
									useRandomStart={useRandomStart}
									lsbBits={lsbBits}
									onKeyChange={setKey}
									onEncryptionChange={setUseEncryption}
									onRandomStartChange={setUseRandomStart}
									onLsbBitsChange={setLsbBits}
								/>
							</div>
						</div>

						{/* Action Button */}
						<div className="mt-8 flex justify-center">
							<button
								onClick={operation === "insert" ? handleInsert : handleExtract}
								disabled={loading}
								className={`px-8 py-3 rounded-lg font-medium text-lg transition-colors ${
									loading
										? "bg-zinc-600 text-zinc-400 cursor-not-allowed"
										: "bg-white text-black hover:bg-zinc-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-zinc-400"
								}`}
							>
								{loading ? (
									<div className="flex items-center">
										<svg
											className="animate-spin -ml-1 mr-3 h-5 w-5 text-zinc-400"
											xmlns="http://www.w3.org/2000/svg"
											fill="none"
											viewBox="0 0 24 24"
										>
											<circle
												className="opacity-25"
												cx="12"
												cy="12"
												r="10"
												stroke="currentColor"
												strokeWidth="4"
											></circle>
											<path
												className="opacity-75"
												fill="currentColor"
												d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
											></path>
										</svg>
										Processing...
									</div>
								) : (
									`${operation === "insert" ? "Insert Message" : "Extract Message"}`
								)}
							</button>
						</div>

						{/* Results */}
						{result && (
							<ResultDisplay
								success={result.success}
								message={result.message}
								psnr={result.psnr}
								downloadUrl={result.downloadUrl}
								filename={result.filename}
								onReset={resetForm}
							/>
						)}
					</div>
				</div>
			</div>
		</div>
	);
}

export default App;
