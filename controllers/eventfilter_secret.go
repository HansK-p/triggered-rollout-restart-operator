package controllers

/*
func findCRsMatchingSecret(logger logr.Logger, cli client.Client, secret *corev1.Secret) ([]*reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger, error) {
	crs := []*reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger{}

	logger = logger.WithValues("type", secret.Type, "CRKind", "reloadrestarttriggerv1alpha1.ResourceReloadRestartTrigger")

	// Check if there is a at least one ResourceReloadRestartTrigger in the same namespace
	instances := reloadrestarttriggerv1alpha1.ResourceReloadRestartTriggerList{}
	err := cli.List(context.TODO(), &instances, client.InNamespace(secret.Namespace))
	if err != nil {
		logger.V(0).Error(err, "Error returned when listing CRs in the secret namespace")
		return crs, err
	}

	if len(instances.Items) == 0 {
		logger.V(2).Info("There are no CRs in the same namespace as the secret so this event is not of interest")
		return crs, nil
	}

	// Check if any of the returned CRs are relevant for the secret
	for idx := range instances.Items {
		instance := &instances.Items[idx]
		logger := logger.WithValues("CRName", instance.Name)
		logger.V(2).Info("Checking the instance")
		for _, trigger := range instance.Spec.Triggers {
			logger := logger.WithValues("Trigger.Kind", trigger.Kind, "Trigger.Name", trigger.Name)
			if trigger.Kind == "Secret" && trigger.Name == secret.Name {
				logger.V(1).Info("This CR has the Secret as a trigger")
				crs = append(crs, instance)
			} else {
				logger.V(1).Info("This CR doesn not have the Secret as a trigger")
			}
		}
	}
	return crs, nil
}

func filterEvents2(logger logr.Logger, cli client.Client, eventObj interface{}) bool {
	logger.V(3).Info("Checking Event Object")
	eventObject, ok := eventObj.(*eventObject)
	if !ok {
		return false
	}
	crs, err := findCRsMatchingSecret(logger, cli, eventObject)
	if err != nil {
		logger.V(0).Error(err, "There was an error finding matching secret source CRs")
		return false
	}
	if len(crs) > 0 {
		logger.V(1).Info("The secret matched at least one source rule. Handle it.", "matches", len(crs))
		return true
	}
	logger.V(2).Info("The secret did not match any source CR. Skip it.")
	return false
}

// Only events matching ResourceReloadRestartCR will result in a reconciler event
func secretEventFilter(logger logr.Logger, cli client.Client) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			logger := logger.WithValues("Event", "Create", "Namespace", e.Object.GetNamespace(), "Name", e.Object.GetName())
			return filterSecretEvents(logger, cli, e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			logger := logger.WithValues("Event", "Delete", "Namespace", e.Object.GetNamespace(), "Name", e.Object.GetName())
			return filterSecretEvents(logger, cli, e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			logger := logger.WithValues("Event", "Update", "Namespace", e.ObjectNew.GetNamespace(), "Name", e.ObjectNew.GetName())
			return filterSecretEvents(logger, cli, e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			logger := logger.WithValues("Event", "Generic", "Namespace", e.Object.GetNamespace(), "Name", e.Object.GetName())
			return filterSecretEvents(logger, cli, e.Object)
		},
	}
}

func secretEventHandler(logger logr.Logger, cli client.Client) handler.MapFunc {
	return func(a client.Object) []reconcile.Request {
		logger := logger.WithValues(
			"function", "secretEventHandler",
			"namespace", a.GetNamespace(),
			"name", a.GetName(),
			"Kind", "Secret",
		)
		logger.V(3).Info("Do a recounciler fan-out for the event")
		secret := &corev1.Secret{}
		secret.Namespace = a.GetNamespace()
		secret.Name = a.GetName()

		requests := []reconcile.Request{}
		return requests
	}
}
*/
