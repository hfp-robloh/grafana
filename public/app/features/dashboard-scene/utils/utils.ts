import { sceneGraph, SceneObject, VizPanel } from '@grafana/scenes';

export function getVizPanelKeyForPanelId(panelId: number) {
  return `panel-${panelId}`;
}

export function getPanelIdForVizPanel(panel: VizPanel): number {
  return parseInt(panel.state.key!.replace('panel-', ''), 10);
}

/**
 * This will also  try lookup based on panelId
 */
export function findVizPanelByKey(scene: SceneObject, key: string | undefined): VizPanel | null {
  if (!key) {
    return null;
  }

  const panel = findVizPanelInternal(scene, key);
  if (panel) {
    return panel;
  }

  // Also try to find by panel id
  const id = parseInt(key, 10);
  if (isNaN(id)) {
    return null;
  }

  return findVizPanelInternal(scene, getVizPanelKeyForPanelId(id));
}

function findVizPanelInternal(scene: SceneObject, key: string | undefined): VizPanel | null {
  if (!key) {
    return null;
  }

  const panel = sceneGraph.findObject(scene, (obj) => obj.state.key === key);
  if (panel) {
    if (panel instanceof VizPanel) {
      return panel;
    } else {
      throw new Error(`Found panel with key ${key} but it was not a VizPanel`);
    }
  }

  return null;
}

/**
 * Force re-render children. This is useful in some edge case scenarios when
 * children deep down the scene graph needs to be re-rendered when some parent state change.
 *
 * Example could be isEditing bool flag or a layout IsDraggable state flag.
 *
 * @param model The model whose children should be re-rendered. It does not force render this model, only the children.
 * @param recursive if it should keep force rendering down to leaf nodess
 */
export function forceRenderChildren(model: SceneObject, recursive?: boolean) {
  model.forEachChild((child) => {
    if (!child.isActive) {
      return;
    }

    child.forceRender();
    forceRenderChildren(child, recursive);
  });
}
