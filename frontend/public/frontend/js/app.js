import { router } from "./router.js";
import { session } from "./services/store.js";
import { renderIcons } from "./utils/helpers.js";

import { LoginPage } from "./pages/login.js";
import { RegisterPage } from "./pages/register.js";
import { DashboardPage } from "./pages/dashboard.js";
import { InventoryPage } from "./pages/inventory.js";
import { MovementsPage } from "./pages/movements.js";
import { ClassesPage } from "./pages/classes.js";
import { DepositsPage } from "./pages/deposits.js";
import { ReportsPage } from "./pages/reports.js";
import { ProfilePage } from "./pages/profile.js";
import { SettingsPage } from "./pages/settings.js";
import { ExportsPage } from "./pages/exports.js";
import { UsersPage } from "./pages/users.js";
import { SupportPage } from "./pages/support.js";
import { ChooseClassPage } from "./pages/choose-class.js";

// Public
router.register("/login", LoginPage, { public: true });
router.register("/register", RegisterPage, { public: true });

// Authenticated
router.register("/dashboard", DashboardPage);
router.register("/inventory", InventoryPage);
router.register("/movements", MovementsPage);
router.register("/classes", ClassesPage);
router.register("/deposits", DepositsPage);
router.register("/reports", ReportsPage);
router.register("/users", UsersPage);
router.register("/settings", SettingsPage);
router.register("/profile", ProfilePage);
router.register("/exports", ExportsPage);
router.register("/support", SupportPage);
router.register("/choose-class", ChooseClassPage);

router.setGuard((route, path) => {
  const isPublic = !!route?.public;
  const authed = session.isAuthenticated();
  if (!authed && !isPublic) return { redirect: "/login" };
  if (authed && isPublic) return { redirect: "/dashboard" };

  // Escolha de turma: um professor vinculado a mais de uma turma precisa
  // escolher com qual trabalhar antes de acessar qualquer tela de dados
  // (protege contra acesso direto por URL sem passar pela escolha).
  if (authed && session.user?.role === "professor" && path !== "/choose-class") {
    const classCount = session.user?.classes?.length || 0;
    if (classCount > 1 && !session.classId) {
      return { redirect: "/choose-class" };
    }
  }
  return null;
});

function boot() {
  if (!location.hash) {
    location.hash = session.isAuthenticated() ? "/dashboard" : "/login";
  } else {
    router.resolve();
  }
  renderIcons();
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", boot);
} else {
  boot();
}

window.addEventListener("load", () => renderIcons());
