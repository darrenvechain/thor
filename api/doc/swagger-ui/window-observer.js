const mutationObserver = new MutationObserver(() => {
    const sections = document.getElementsByClassName('opblock-tag-section is-open')

    for (let i = 0; i < sections.length; i++) {
        //Find a child in each section with id="operations-tag-Subscriptions"
        const section = sections[i]

        const child = section.querySelector('[id^="operations-tag-Subscriptions"]')

        if (child) {
            const classesToRemove = [
                "response-col_description__inner",
                "responses-header",
                "try-out",
                "response-col_status",
                "response-col_links",
                "response-controls",
            ]

            classesToRemove.forEach((className) => {
                const elements = section.querySelectorAll(`[class^="${className}"]`)
                elements.forEach((el) => el.remove())
            })
        }
    }
})

mutationObserver.observe(document.body, {attributes: false, childList: true, characterData: false, subtree:true});
