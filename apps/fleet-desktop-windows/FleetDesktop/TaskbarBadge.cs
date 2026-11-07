using System;
using System.Globalization;
using System.Windows;
using System.Windows.Media;
using System.Windows.Media.Imaging;

namespace FleetDesktop;

/// <summary>
/// Renders a numbered overlay icon for the taskbar. Equivalent of macOS's
/// <c>NSApp.dockTile.badgeLabel</c> — a small red circle with the failing-policy count.
/// </summary>
internal static class TaskbarBadge
{
    public static ImageSource Create(int count)
    {
        const int size = 16;
        var text = count > 99 ? "99+" : count.ToString(CultureInfo.InvariantCulture);

        var visual = new DrawingVisual();
        using (var dc = visual.RenderOpen())
        {
            var center = new Point(size / 2.0, size / 2.0);

            // Subtle dark border for legibility on light taskbars.
            dc.DrawEllipse(
                Brushes.Black,
                new Pen(Brushes.Black, 1),
                center,
                size / 2.0,
                size / 2.0);

            // Red fill on top of the border.
            var red = (Color)ColorConverter.ConvertFromString("#E5392A");
            dc.DrawEllipse(
                new SolidColorBrush(red),
                null,
                center,
                (size / 2.0) - 1,
                (size / 2.0) - 1);

            // Pick a font size that fits — text up to 3 chars ("99+") at 9pt fits inside a 16px circle.
            var fontSize = text.Length switch
            {
                1 => 11.0,
                2 => 9.0,
                _ => 7.0,
            };

            var typeface = new Typeface(
                new FontFamily("Segoe UI"),
                FontStyles.Normal,
                FontWeights.Bold,
                FontStretches.Normal);

            var ft = new FormattedText(
                text,
                CultureInfo.InvariantCulture,
                FlowDirection.LeftToRight,
                typeface,
                fontSize,
                Brushes.White,
                pixelsPerDip: 1.0);

            var origin = new Point(
                center.X - (ft.Width / 2),
                center.Y - (ft.Height / 2));
            dc.DrawText(ft, origin);
        }

        var rtb = new RenderTargetBitmap(size, size, 96, 96, PixelFormats.Pbgra32);
        rtb.Render(visual);
        rtb.Freeze();
        return rtb;
    }
}
