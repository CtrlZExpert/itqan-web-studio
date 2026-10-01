const form = document.querySelector(".contact-form")
const formStatus = document.querySelector(".form-status")


form.addEventListener("submit", async (event) => {
	event.preventDefault();
	formStatus.textContent = "Sending your message...";
	try {
		const formData = {
			name: form.name.value,
			email: form.email.value,
			message: form.message.value,
		}
		const response = await fetch("http://localhost:8080/api/contact", {
			method: "POST",
			headers: {
				"Content-Type": "application/json",
			},
			body: JSON.stringify(formData),
		});
		if (response.ok) {
			formStatus.textContent = "Thanks for reaching out! We'll be in touch soon.";
			form.reset();
		} else if (response.status === 429) {
			formStatus.textContent = "Too many submission. Please wait 10 minutes and try again"
		} else {

			formStatus.textContent = "Something went wrong. Please try again or email us directly at itqanwebstudio@gmail.com";
		}

	} catch (error) {

		formStatus.textContent = "Something went wrong. Please try again or email us directly at itqanwebstudio@gmail.com";
	}
});
