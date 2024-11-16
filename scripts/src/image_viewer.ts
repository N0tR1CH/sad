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
    const croppedCanvas: HTMLCanvasElement = this.cropper.getCroppedCanvas();
    this.initCropper();
    const imgDataUrl = croppedCanvas.toDataURL("image/webp");
    this.imageUrl = imgDataUrl;
    croppedCanvas.toBlob((blob) => {
      const fileInput = this.$refs.fileInput as HTMLInputElement;
      const dataTransfer = new DataTransfer();
      const newFile = new File([blob], fileInput.files[0].name, {
        type: "image/webp",
      });
      dataTransfer.items.add(newFile);
      fileInput.files = dataTransfer.files;
    }, "image/webp");
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
      input.files[0] = null;
      return;
    }
    const reader: FileReader = new FileReader();

    reader.readAsDataURL(file);
    reader.onload = (onLoadEvent: ProgressEvent<FileReader>) => {
      const res = onLoadEvent.target.result as string;
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
