#include "mainwindow.h"
#include "ui_mainwindow.h"

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , ui(new Ui::MainWindow)
{
    ui->setupUi(this);
}

MainWindow::~MainWindow()
{
    delete ui;
}

void MainWindow::on_pushButtonOpen_clicked()
{

}


void MainWindow::on_plainTextEditSource_textChanged()
{
    stSource_ = ui->plainTextEditSource->toPlainText();
}

