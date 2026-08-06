"use client";

import { useState, useCallback } from "react";
import { useDropzone } from "react-dropzone";
import { Upload, FileText, X, Check, Loader2, File } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { DocumentHelpers } from "@/lib/models/document.model";

interface DocumentUploadProps {
  onUpload: (file: File, title: string) => Promise<void>;
  isUploading?: boolean;
}

export function DocumentUpload({
  onUpload,
  isUploading = false,
}: DocumentUploadProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [title, setTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const onDrop = useCallback(
    (acceptedFiles: File[], rejectedFiles: unknown[]) => {
      setError(null);
      setSuccess(false);

      if (rejectedFiles.length > 0) {
        setError("Only PDF files are accepted");
        return;
      }

      if (acceptedFiles.length > 0) {
        const file = acceptedFiles[0];
        setSelectedFile(file);
        const nameWithoutExt = file.name.replace(/\.pdf$/i, "");
        setTitle(nameWithoutExt);
      }
    },
    []
  );

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      "application/pdf": [".pdf"],
    },
    maxFiles: 1,
    disabled: isUploading,
  });

  const handleUpload = async () => {
    if (!selectedFile || !title.trim()) return;

    setError(null);
    try {
      await onUpload(selectedFile, title.trim());
      setSuccess(true);
      setSelectedFile(null);
      setTitle("");
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    }
  };

  const clearSelection = () => {
    setSelectedFile(null);
    setTitle("");
    setError(null);
  };

  // If we have a file, show the confirmation/title form
  if (selectedFile) {
      return (
        <div className="space-y-4">
             {error && (
                <Alert variant="destructive" className="border-red-200 bg-red-50">
                <AlertDescription className="text-red-700">{error}</AlertDescription>
                </Alert>
            )}

            <div className="rounded-xl border border-gray-200 bg-white p-4">
                <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                         <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-red-50">
                            <FileText className="h-5 w-5 text-red-500" />
                        </div>
                        <div>
                             <p className="font-medium text-gray-900 line-clamp-1">{selectedFile.name}</p>
                             <p className="text-xs text-gray-500">{DocumentHelpers.formatFileSize(selectedFile.size)}</p>
                        </div>
                    </div>
                    <Button variant="ghost" size="icon" onClick={clearSelection} disabled={isUploading}>
                        <X className="h-4 w-4 text-gray-500" />
                    </Button>
                </div>

                <div className="mt-4 space-y-2">
                    <Label htmlFor="doc-title">Document Title</Label>
                    <Input
                        id="doc-title"
                        value={title}
                        onChange={(e) => setTitle(e.target.value)}
                        placeholder="My Document"
                        disabled={isUploading}
                    />
                </div>
            </div>
            
            <div className="flex gap-2">
                 <Button variant="outline" className="flex-1" onClick={clearSelection} disabled={isUploading}>
                    Cancel
                 </Button>
                 <Button 
                    className="flex-1 bg-gray-900 text-white hover:bg-gray-800" 
                    onClick={handleUpload}
                    disabled={isUploading || !title.trim()}
                >
                    {isUploading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Upload className="mr-2 h-4 w-4" />}
                    {isUploading ? "Uploading..." : "Upload PDF"}
                 </Button>
            </div>
        </div>
      )
  }

  return (
    <div className="space-y-4">
      {success && (
        <Alert className="border-emerald-200 bg-emerald-50 mb-4">
          <Check className="h-4 w-4 text-emerald-600" />
          <AlertDescription className="text-emerald-700">
            Document uploaded successfully!
          </AlertDescription>
        </Alert>
      )}

      <div
        {...getRootProps()}
        className={cn(
          "flex flex-col items-center justify-center rounded-2xl border-2 border-dashed p-10 transition-all duration-200 cursor-pointer",
          isDragActive
            ? "border-gray-900 bg-gray-50"
            : "border-gray-200 bg-gray-50/50 hover:border-gray-300 hover:bg-gray-50",
          isUploading && "cursor-not-allowed opacity-60"
        )}
      >
        <input {...getInputProps()} />
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 mb-4">
          <Upload className="h-6 w-6 text-gray-400" />
        </div>
        <p className="text-sm font-medium text-gray-900 text-center">
          {isDragActive ? "Drop PDF here" : "Click or drag PDF to upload"}
        </p>
        <p className="mt-1 text-xs text-gray-500 text-center">
            Max file size 10MB
        </p>
      </div>
    </div>
  );
}

