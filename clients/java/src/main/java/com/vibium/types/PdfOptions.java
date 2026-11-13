package com.vibium.types;

import java.util.List;

/**
 * Options for printing a page to PDF. Unset options keep the browser's print
 * defaults (portrait, scale 1, 1cm margins, no background, letter-size page,
 * all pages, shrink to fit). Margins and page size are in cm.
 */
public class PdfOptions {
    private Boolean landscape;
    private Double scale;
    private Boolean background;
    private Double marginTop;
    private Double marginBottom;
    private Double marginLeft;
    private Double marginRight;
    private Double pageWidth;
    private Double pageHeight;
    private List<Object> pageRanges;
    private Boolean shrinkToFit;

    public PdfOptions landscape(boolean landscape) { this.landscape = landscape; return this; }
    public PdfOptions scale(double scale) { this.scale = scale; return this; }
    public PdfOptions background(boolean background) { this.background = background; return this; }
    public PdfOptions marginTop(double cm) { this.marginTop = cm; return this; }
    public PdfOptions marginBottom(double cm) { this.marginBottom = cm; return this; }
    public PdfOptions marginLeft(double cm) { this.marginLeft = cm; return this; }
    public PdfOptions marginRight(double cm) { this.marginRight = cm; return this; }
    /** Set all four margins at once, in cm. */
    public PdfOptions margin(double cm) {
        return marginTop(cm).marginBottom(cm).marginLeft(cm).marginRight(cm);
    }
    public PdfOptions pageWidth(double cm) { this.pageWidth = cm; return this; }
    public PdfOptions pageHeight(double cm) { this.pageHeight = cm; return this; }
    /** Pages to print: integers and range strings, e.g. List.of(1, "3-5"). */
    public PdfOptions pageRanges(List<Object> pageRanges) { this.pageRanges = pageRanges; return this; }
    public PdfOptions shrinkToFit(boolean shrinkToFit) { this.shrinkToFit = shrinkToFit; return this; }

    public Boolean landscape() { return landscape; }
    public Double scale() { return scale; }
    public Boolean background() { return background; }
    public Double marginTop() { return marginTop; }
    public Double marginBottom() { return marginBottom; }
    public Double marginLeft() { return marginLeft; }
    public Double marginRight() { return marginRight; }
    public Double pageWidth() { return pageWidth; }
    public Double pageHeight() { return pageHeight; }
    public List<Object> pageRanges() { return pageRanges; }
    public Boolean shrinkToFit() { return shrinkToFit; }
}
