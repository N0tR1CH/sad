type ImageViewerCallback = (src: string) => void;

export default () => ({
  imageUrl: "",
  fileToDataUrl(e: Event, callback: ImageViewerCallback) {
    const input = e.target as HTMLInputElement;
    if (!input.files.length) {
      return;
    }
    const file: File = input.files[0];
    const sizeMb = file.size / (1024 * 1024);
    if (sizeMb >= 2) {
      window.Swal.fire({
        icon: "error",
        title: "File too large",
        text: "File must be under 2mbs",
      });
      input.value = "";
      return;
    }
    const reader: FileReader = new FileReader();

    reader.readAsDataURL(file);
    reader.onload = (onLoadEvent: ProgressEvent<FileReader>) => {
      const res: string = onLoadEvent.target.result as string;
      callback(res);
    };
  },
  fileChosen(e: Event) {
    this.fileToDataUrl(e, (src: string) => {
      this.imageUrl = src;
    });
  },
});
