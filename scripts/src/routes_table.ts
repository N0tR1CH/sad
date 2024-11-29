export default () => ({
  init() {
    console.log("Routes Table initialized!");
  },
  routes: [],
  checkboxEls: [],
  checkAll() {
    if (this.checkboxEls.length === 0) {
      this.checkboxEls = this.$refs.tbody.querySelectorAll("input");
    }
    for (let i = 0; i < this.checkboxEls.length; i++) {
      this.routes.push(this.checkboxEls[i].value);
    }
  },
  unCheckAll() {
    if (this.checkboxEls.length === 0) {
      this.checkboxEls = this.$refs.tbody.querySelectorAll("input");
    }
    for (let i = 0; i < this.checkboxEls.length; i++) {
      this.routes = [];
    }
  },
  copyToClipboard() {
    const jsonArr = [];
    this.routes.map((el) => {
      const jsonObject = JSON.parse(el);
      jsonArr.push(jsonObject);
    });
    const stringifiedJson = JSON.stringify(jsonArr);
    navigator.clipboard.writeText(stringifiedJson);
    const Toast = Swal.mixin({
      toast: true,
      position: "top-end",
      showConfirmButton: false,
      timer: 3000,
      timerProgressBar: true,
      didOpen: (toast) => {
        toast.onmouseenter = Swal.stopTimer;
        toast.onmouseleave = Swal.resumeTimer;
      },
    });
    Toast.fire({
      icon: "success",
      title: "Json copied to clipboard",
    });
  },
});
