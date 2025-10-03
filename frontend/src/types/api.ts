export interface StegoRequest {
    key: string;
    use_encryption: boolean;
    use_random_start: boolean;
    lsb_bits: number;
    secret_filename?: string;
}

export interface StegoResponse {
    success: boolean;
    message: string;
    psnr?: number;
    file_blob?: Blob;
    download_url?: string;
    filename?: string;
}

export interface ExtractRequest {
    key: string;
    use_encryption: boolean;
    use_random_start: boolean;
    lsb_bits: number;
}

export interface ExtractResponse {
    success: boolean;
    message: string;
    file_blob?: Blob;
    download_url?: string;
    filename?: string;
}

export interface HealthResponse {
    status: string;
    service: string;
    version: string;
}
