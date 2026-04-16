import React, { useContext, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";

import notificationsAPI from "services/entities/notifications";
import { NotificationContext } from "context/notification";
import {
  CATEGORY_LABELS,
  INotificationPreference,
  IListNotificationPreferencesResponse,
  NotificationCategory,
} from "interfaces/notification_preferences";

// @ts-ignore
import Slider from "components/forms/fields/Slider";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "notification-settings";

const PREFERENCES_QUERY_KEY = ["notification-preferences"] as const;

const NotificationSettings = (): JSX.Element => {
  const queryClient = useQueryClient();
  const { renderFlash } = useContext(NotificationContext);

  const { data, isLoading, isError } = useQuery<
    IListNotificationPreferencesResponse,
    Error
  >(PREFERENCES_QUERY_KEY, () => notificationsAPI.getPreferences(), {
    refetchOnWindowFocus: false,
  });

  // Index the full preference grid by category so a toggle click can flip the
  // in_app row without disturbing the other channels' rows.
  const byCategory = useMemo(() => {
    const map = new Map<NotificationCategory, INotificationPreference[]>();
    (data?.preferences ?? []).forEach((p) => {
      const list = map.get(p.category) ?? [];
      list.push(p);
      map.set(p.category, list);
    });
    return map;
  }, [data]);

  const updateMutation = useMutation(
    (prefs: INotificationPreference[]) =>
      notificationsAPI.updatePreferences(prefs),
    {
      // Optimistic update: replace the cache with the outgoing prefs so the
      // toggle flips immediately. If the server rejects, we invalidate on
      // error to roll back to the true state.
      onMutate: async (prefs) => {
        await queryClient.cancelQueries(PREFERENCES_QUERY_KEY);
        const prev = queryClient.getQueryData<IListNotificationPreferencesResponse>(
          PREFERENCES_QUERY_KEY
        );
        queryClient.setQueryData<IListNotificationPreferencesResponse>(
          PREFERENCES_QUERY_KEY,
          { preferences: prefs }
        );
        return { prev };
      },
      onError: (_err, _vars, ctx) => {
        if (ctx?.prev) {
          queryClient.setQueryData(PREFERENCES_QUERY_KEY, ctx.prev);
        }
        renderFlash("error", "Could not update notification preferences.");
      },
      onSuccess: (res) => {
        // Server is the authority; replace cache with its response.
        queryClient.setQueryData(PREFERENCES_QUERY_KEY, res);
      },
    }
  );

  const toggleInApp = (category: NotificationCategory, currentlyEnabled: boolean) => {
    const all = data?.preferences ?? [];
    const next = all.map((p) =>
      p.category === category && p.channel === "in_app"
        ? { ...p, enabled: !currentlyEnabled }
        : p
    );
    updateMutation.mutate(next);
  };

  if (isLoading) return <Spinner />;
  if (isError) return <DataError />;

  return (
    <div className={baseClass}>
      <h2>Notifications</h2>
      <p className={`${baseClass}__description`}>
        Choose the types of notifications that should appear in your in-app
        notification center.
      </p>
      <ul className={`${baseClass}__list`}>
        {Object.entries(CATEGORY_LABELS).map(([categoryKey, label]) => {
          const category = categoryKey as NotificationCategory;
          const prefs = byCategory.get(category) ?? [];
          const inAppPref = prefs.find((p) => p.channel === "in_app");
          const enabled = inAppPref?.enabled ?? true;
          return (
            <li key={category} className={`${baseClass}__item`}>
              <Slider
                value={enabled}
                onChange={() => toggleInApp(category, enabled)}
                inactiveText="Off"
                activeText="On"
                disabled={updateMutation.isLoading}
              />
              <div className={`${baseClass}__item-text`}>
                <div className={`${baseClass}__item-title`}>{label.title}</div>
                <div className={`${baseClass}__item-description`}>
                  {label.description}
                </div>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
};

export default NotificationSettings;
