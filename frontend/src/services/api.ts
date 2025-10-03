import axios from "axios";
import { StegoResponse, ExtractResponse, HealthResponse } from "../types/api";

const API_BASE_URL = process.env.REACT_APP_API_URL || "http://localhost:8080";

const api = axios.create({
	baseURL: API_BASE_URL,
	timeout: 30000, // 30 seconds timeout for file uploads
});

export const steganographyAPI = {
	// Health check
	healthCheck: async (): Promise<HealthResponse> => {
		const response = await api.get("/api/v1/health");
		return response.data;
	},

	// Insert secret message into audio
	insertMessage: async (
		audioFile: File,
		secretFile: File,
		key: string,
		useEncryption: boolean,
		useRandomStart: boolean,
		lsbBits: number,
	): Promise<StegoResponse> => {
		try {
			const formData = new FormData();
			formData.append("audio_file", audioFile);
			formData.append("secret_file", secretFile);
			formData.append("key", key);
			formData.append("use_encryption", useEncryption.toString());
			formData.append("use_random_start", useRandomStart.toString());
			formData.append("lsb_bits", lsbBits.toString());

			const response = await api.post("/api/v1/stego/insert", formData, {
				headers: {
					"Content-Type": "multipart/form-data",
				},
				responseType: "blob", // Expect binary response
			});

			// Extract headers
			const psnr = response.headers["x-stego-psnr"];
			const message = response.headers["x-stego-message"];

			// Create blob URL for download
			const blob = new Blob([response.data], { type: "audio/mpeg" });
			const downloadUrl = URL.createObjectURL(blob);

			return {
				success: true,
				message: message || "File processed successfully",
				psnr: psnr ? parseFloat(psnr) : undefined,
				file_blob: blob,
				download_url: downloadUrl,
				filename: `${audioFile.name.split(".")[0]}_stego.mp3`,
			};
		} catch (error: any) {
			// Handle blob error responses
			if (error.response && error.response.data instanceof Blob) {
				const text = await error.response.data.text();
				try {
					const errorData = JSON.parse(text);
					throw new Error(
						errorData.error || errorData.message || "Processing failed",
					);
				} catch {
					throw new Error(text || "Processing failed");
				}
			}
			throw error;
		}
	},

	// Extract secret message from stego audio
	extractMessage: async (
		stegoAudioFile: File,
		key: string,
		useEncryption: boolean,
		useRandomStart: boolean,
		lsbBits: number,
	): Promise<ExtractResponse> => {
		try {
			const formData = new FormData();
			formData.append("stego_file", stegoAudioFile);
			formData.append("key", key);
			formData.append("use_encryption", useEncryption.toString());
			formData.append("use_random_start", useRandomStart.toString());
			formData.append("lsb_bits", lsbBits.toString());

			const response = await api.post("/api/v1/stego/extract", formData, {
				headers: {
					"Content-Type": "multipart/form-data",
				},
				responseType: "blob", // Expect binary response
			});

			// Extract filename from Content-Disposition header
			const contentDisposition = response.headers["content-disposition"];
			let filename = "extracted_file";
			
			if (contentDisposition) {
				// Match filename= followed by optional quotes and capture the filename
				// Handles: filename=file.pdf, filename="file.pdf", attachment; filename=file.pdf
				const filenameMatch = contentDisposition.match(/filename\s*=\s*"?([^";\r\n]+)"?/i);
				if (filenameMatch) {
					filename = filenameMatch[1].trim();
				}
			}

			// Create blob URL for download
			const blob = new Blob([response.data], {
				type: "application/octet-stream",
			});
			const downloadUrl = URL.createObjectURL(blob);

			return {
				success: true,
				message: "Secret file extracted successfully",
				file_blob: blob,
				download_url: downloadUrl,
				filename: filename,
			};
		} catch (error: any) {
			// Handle blob error responses
			if (error.response && error.response.data instanceof Blob) {
				const text = await error.response.data.text();
				try {
					const errorData = JSON.parse(text);
					throw new Error(
						errorData.error || errorData.message || "Extraction failed",
					);
				} catch {
					throw new Error(text || "Extraction failed");
				}
			}
			throw error;
		}
	},

	// Utility function to clean up blob URLs
	cleanupBlobUrl: (url: string) => {
		URL.revokeObjectURL(url);
	},
};

export default api;
