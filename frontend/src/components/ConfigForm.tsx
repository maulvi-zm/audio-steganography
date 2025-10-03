import React from "react";

interface ConfigFormProps {
    stringKey: string;
    useEncryption: boolean;
    useRandomStart: boolean;
    lsbBits: number;
    onKeyChange: (key: string) => void;
    onEncryptionChange: (value: boolean) => void;
    onRandomStartChange: (value: boolean) => void;
    onLsbBitsChange: (value: number) => void;
}

const ConfigForm: React.FC<ConfigFormProps> = ({
    stringKey,
    useEncryption,
    useRandomStart,
    lsbBits,
    onKeyChange,
    onEncryptionChange,
    onRandomStartChange,
    onLsbBitsChange,
}) => {
    return (
        <div className="space-y-4">
            <div>
                <label
                    htmlFor="stegoKey"
                    className="block text-sm font-medium text-zinc-300 mb-2"
                >
                    Stego Key <span className="text-red-500">*</span>
                </label>
                <input
                    type="text"
                    id="stegoKey"
                    value={stringKey}
                    onChange={(e) => {
                        console.log(e.target.value);
                        onKeyChange(e.target.value);
                    }}
                    className="w-full px-3 py-2 bg-zinc-900 border border-zinc-600 text-white rounded-md shadow-sm focus:outline-none focus:ring-zinc-400 focus:border-zinc-400"
                    placeholder="Enter your stego key"
                    required
                />
                <p className="text-xs text-zinc-400 mt-1">
                    Used for Vigenère cipher and random position generation
                </p>
            </div>

            <div>
                <label
                    htmlFor="lsbBits"
                    className="block text-sm font-medium text-zinc-300 mb-2"
                >
                    LSB Bits (1-4) <span className="text-red-500">*</span>
                </label>
                <select
                    id="lsbBits"
                    value={lsbBits}
                    onChange={(e) => onLsbBitsChange(parseInt(e.target.value))}
                    className="w-full px-3 py-2 bg-zinc-900 border border-zinc-600 text-white rounded-md shadow-sm focus:outline-none focus:ring-zinc-400 focus:border-zinc-400"
                >
                    <option value={1}>
                        1 bit (highest quality, lowest capacity)
                    </option>
                    <option value={2}>2 bits</option>
                    <option value={3}>3 bits</option>
                    <option value={4}>
                        4 bits (lowest quality, highest capacity)
                    </option>
                </select>
                <p className="text-xs text-zinc-400 mt-1">
                    More bits = higher capacity but lower audio quality
                </p>
            </div>

            <div className="space-y-3">
                <div className="flex items-center">
                    <input
                        type="checkbox"
                        id="useEncryption"
                        checked={useEncryption}
                        onChange={(e) => onEncryptionChange(e.target.checked)}
                        className="h-4 w-4 text-zinc-400 focus:ring-zinc-400 bg-zinc-900 border-zinc-600 rounded"
                    />
                    <label
                        htmlFor="useEncryption"
                        className="ml-2 text-sm text-zinc-300"
                    >
                        Use Extended Vigenère Cipher encryption
                    </label>
                </div>

                <div className="flex items-center">
                    <input
                        type="checkbox"
                        id="useRandomStart"
                        checked={useRandomStart}
                        onChange={(e) => onRandomStartChange(e.target.checked)}
                        className="h-4 w-4 text-zinc-400 focus:ring-zinc-400 bg-zinc-900 border-zinc-600 rounded"
                    />
                    <label
                        htmlFor="useRandomStart"
                        className="ml-2 text-sm text-zinc-300"
                    >
                        Use random insertion positions
                    </label>
                </div>
            </div>
        </div>
    );
};

export default ConfigForm;
