// Fields that are managed by OpenSOHO (see app.opensoho.isCollectionFieldReadOnly)
// are shown read-only in the record upsert modal.

function readOnlyInput(props, fieldType) {
    return t.div(
        { className: `record-field-input field-type-${props.field.type} opensoho-readonly-field` },
        t.div(
            { className: "fields" },
            t.div(
                { className: "field" },
                t.label(
                    null,
                    t.i({ ariaHidden: true, className: fieldType.icon || "" }),
                    t.span({ className: "txt" }, props.field.name),
                    t.i({
                        ariaHidden: true,
                        className: "ri-lock-line txt-hint",
                        ariaDescription: app.attrs.tooltip("Managed by OpenSOHO"),
                    }),
                ),
                t.div(
                    { className: "readonly-value" },
                    fieldType.view
                        ? fieldType.view({
                            get record() {
                                return props.record;
                            },
                            get field() {
                                return props.field;
                            },
                            short: false,
                        })
                        : t.span(null, () => String(props.record[props.field.name] ?? "")),
                ),
            ),
        ),
    );
}

for (const fieldType of Object.values(app.fieldTypes)) {
    if (!fieldType.input) {
        continue;
    }

    const originalInput = fieldType.input;
    fieldType.input = (props) => {
        if (
            !props.field.primaryKey
            && app.opensoho.isCollectionFieldReadOnly(props.collection?.name, props.field.name)
        ) {
            return readOnlyInput(props, fieldType);
        }

        return originalInput(props);
    };
}
