import Cropper from "cropperjs";

type ImageViewerCallback = (src: string) => void;

export default () => ({
  init() {
    console.log("ImageViewer initialized!");
  },
  initCropper() {
    const image = this.$refs.image as HTMLImageElement;
    if (!image) {
      console.error("Image element not found");
      return;
    }

    // Destroy previous instance
    if (this.cropper) {
      this.cropper.destroy();
    }

    // Initialize new cropper after image loads
    image.onload = () => {
      this.cropper = new Cropper(image, {
        aspectRatio: 1,
        viewMode: 1,
        background: false,
        ready: () => {
          this.croppable = true;
        },
      });
    };

    console.log("cropper initialized");
  },
  cropper: null,
  imageUrl: "",
  croppable: false,
  cropImg(e: Event) {
    e.preventDefault();
    if (!this.croppable) {
      return;
    }
    const croppedCanvas = this.cropper.getCroppedCanvas();
    this.initCropper();
    const imgDataUrl = croppedCanvas.toDataURL("image/webp");
    this.imageUrl = imgDataUrl;
  },
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
    this.initCropper();
  },
  fileChosen(e: Event) {
    this.fileToDataUrl(e, (src: string) => {
      this.imageUrl = src;
    });
  },
});
