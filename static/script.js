// validate port number
const portInput = document.getElementById("port");
portInput.addEventListener("input", function () {
  const port = parseInt(this.value);
  const isValidPort = port === 587 || port === 465 || port === 25;
  this.setCustomValidity(
    isValidPort ? "" : "Invalid SMTP port. Enter either 25 or 465 or 587"
  );
});

// toggle upload field
const htmlRadioButton = document.getElementById("type-html");
const htmlFileGroup = document.getElementById("html-file-group");
const htmlFileInput = document.getElementById("html-file");
const plainRadioButton = document.getElementById("type-normal");
const messageTextArea = document.getElementById("message");

htmlRadioButton.addEventListener("change", function () {
  if (this.checked) {
    messageTextArea.required = false;
    htmlFileGroup.style.display = "block";
    htmlFileInput.setAttribute("required", "required");
  }
});

plainRadioButton.addEventListener("change", function () {
  if (this.checked) {
    messageTextArea.required = true;
    htmlFileGroup.style.display = "none";
    htmlFileInput.removeAttribute("required");
  }
});

// show error messages from the server
function showError(message) {
  const errorMessage = document.getElementById("error-message");
  errorMessage.textContent = message;
  errorMessage.style.display = "block";
  errorMessage.style.transform = "translateY(0)";
  setTimeout(hideError, 3000);
}
//parsed to the setTimeout function to hide the Error
function hideError() {
  const errorMessage = document.getElementById("error-message");
  errorMessage.style.transform = "translateY(-400%)";
}

// get all needed elements
const sentCountElement = document.getElementById("sent-count");
const failedCountElement = document.getElementById("failed-count");
const statusCounts = document.querySelector(".status-counts");
const statusDisplayContainer = document.querySelector(".status-container");
const statusDisplay = document.getElementById("status-display");

//establish SSE connection to the Go server
const eventSource = new EventSource("/status");

// process message update from server
eventSource.onmessage = (event) => {
  statusCounts.style.display = "block";
  statusDisplayContainer.style.display = "block";
  const result = JSON.parse(event.data);
  const { email, status } = result;
  const emailStatus = status ? "Sent" : "Failed";
  const statusColor = status ? "green" : "red";
  statusDisplay.innerHTML += `${email} -> <span style="color:${statusColor}">${emailStatus}</span><br>`;

  // increment sent and failed on realtime update from server
  if (status) {
    sentCountElement.textContent = parseInt(sentCountElement.textContent) + 1;
  } else {
    failedCountElement.textContent =
      parseInt(failedCountElement.textContent) + 1;
  }
};

// handle SSE error connection
eventSource.onerror = () => {
  console.error("Error in SSE connection.");
  eventSource.close();
};

// handle closed SSE connection
eventSource.onclose = () => {
  console.log("SSE connection closed.");
  button.disabled = false;
};

// make sure none of the fields are empty
function submitForm() {
  const form = document.getElementById("mail-form");
  if (!form.checkValidity()) {
    form.reportValidity();
    return;
  }

  const button = document.getElementById("submit");
  const buttonText = button.querySelector(".button-text");
  const loader = button.querySelector(".loader");
  buttonText.style.display = "none";
  button.style.padding = "15px";
  loader.style.display = "block";

  sentCountElement.textContent = 0;
  failedCountElement.textContent = 0;
  statusDisplay.innerHTML = "";

  // properly format form input values
  const formData = new FormData(form);
  console.log("button is here!");
  fetch("/", {
    method: "POST",
    body: formData,
  })
    .then((response) => {
      if (response.ok) {
        buttonText.style.display = "inline-block";
        loader.style.display = "none";
        button.style.padding = "10px";
        button.disabled = false;
        response.text().then((data) => {});
      } else if (response.status === 400) {
        response.text().then((data) => {
          showError(data);
          buttonText.style.display = "inline-block";
          loader.style.display = "none";
          button.style.padding = "10px";
          button.disabled = false;
        });
      } else console.log(response);
    })
    .catch((error) => {
      console.error("An error occurred:", error);
    });
}
const button = document.getElementById("submit");
button.addEventListener("click", function (event) {
  event.preventDefault();
  submitForm();
});
